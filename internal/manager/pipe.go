package manager

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/paulbellamy/ratecounter"
)

type pipe struct {
	camp       *models.Campaign
	rate       *ratecounter.RateCounter
	wg         *sync.WaitGroup
	sent       atomic.Int64
	lastID     atomic.Uint64
	errors     atomic.Uint64
	stopped    atomic.Bool
	withErrors atomic.Bool

	// Per-campaign sliding-window state. Solomon fork: the original Listmonk
	// stored slidingCount/slidingStart on the Manager singleton, so every
	// running campaign shared one global N-per-window budget. That meant a
	// noisy campaign could starve every other campaign on the box. Each pipe
	// now tracks its own window so per-campaign caps are real.
	slidingCount int
	slidingStart time.Time

	// done is closed by Stop() so an in-progress sliding-window sleep wakes
	// immediately on pause/cancel instead of blocking the goroutine for up to
	// SlidingWindowDuration. Use stopOnce to guarantee a single close.
	done     chan struct{}
	stopOnce sync.Once

	m *Manager
}

// newPipe adds a campaign to the process queue.
func (m *Manager) newPipe(c *models.Campaign) (*pipe, error) {
	// Validate messenger.
	if _, ok := m.messengers[c.Messenger]; !ok {
		m.store.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return nil, fmt.Errorf("unknown messenger %s on campaign %s", c.Messenger, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		return nil, err
	}

	// Load any media/attachments.
	if err := m.attachMedia(c); err != nil {
		return nil, err
	}

	// Add the campaign to the active map.
	p := &pipe{
		camp:         c,
		rate:         ratecounter.NewRateCounter(time.Minute),
		wg:           &sync.WaitGroup{},
		slidingStart: time.Now(),
		done:         make(chan struct{}),
		m:            m,
	}

	// Increment the waitgroup so that Wait() blocks immediately. This is necessary
	// as a campaign pipe is created first and subscribers/messages under it are
	// fetched asynchronolusly later. The messages each add to the wg and that
	// count is used to determine the exhaustion/completion of all messages.
	p.wg.Add(1)

	go func() {
		// Wait for all the messages in the campaign to be processed
		// (successfully or skipped after errors or cancellation).
		p.wg.Wait()

		p.cleanup()
	}()

	m.pipesMut.Lock()
	// If a stale pipe is still registered for this campaign (e.g. its
	// goroutine is still draining a previous sliding-window sleep), stop it
	// so it can exit cleanly. The stale pipe's deferred cleanup uses a
	// pointer-identity check before deleting the map entry, so it won't
	// remove the new pipe we install on the next line.
	if old, ok := m.pipes[c.ID]; ok {
		old.Stop(false)
	}
	m.pipes[c.ID] = p
	m.pipesMut.Unlock()
	return p, nil
}

// NextSubscribers processes the next batch of subscribers in a given campaign.
// It returns a bool indicating whether any subscribers were processed
// in the current batch or not. A false indicates that all subscribers
// have been processed, or that a campaign has been paused or cancelled.
func (p *pipe) NextSubscribers() (bool, error) {
	// Fetch the next batch of subscribers from a 'running' campaign.
	subs, err := p.m.store.NextSubscribers(p.camp.ID, p.m.cfg.BatchSize)
	if err != nil {
		return false, fmt.Errorf("error fetching campaign subscribers (%s): %v", p.camp.Name, err)
	}

	// There are no subscribers from the query. Either all subscribers on the campaign
	// have been processed, or the campaign has changed from 'running' to 'paused' or 'cancelled'.
	if len(subs) == 0 {
		return false, nil
	}

	// Is there a sliding window limit configured?
	hasSliding := p.m.cfg.SlidingWindow &&
		p.m.cfg.SlidingWindowRate > 0 &&
		p.m.cfg.SlidingWindowDuration.Seconds() > 1

	// Push messages.
	for _, s := range subs {
		msg, err := p.newMessage(s)
		if err != nil {
			p.m.log.Printf("error rendering message (%s) (%s): %v", p.camp.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		p.m.campMsgQ <- msg

		// Check if the sliding window is active. Counter is on the pipe so
		// every campaign gets its own independent N-per-window budget; one
		// hot campaign no longer eats every other campaign's quota.
		if hasSliding {
			diff := time.Since(p.slidingStart)

			// Window has expired. Reset the clock.
			if diff >= p.m.cfg.SlidingWindowDuration {
				p.slidingStart = time.Now()
				p.slidingCount = 0
			}

			// Have the messages exceeded the limit?
			p.slidingCount++
			if p.slidingCount >= p.m.cfg.SlidingWindowRate {
				wait := p.m.cfg.SlidingWindowDuration - diff

				p.m.log.Printf("campaign %q messages exceeded (%d) for the window (%v since %s). Sleeping for %s.",
					p.camp.Name,
					p.slidingCount,
					p.m.cfg.SlidingWindowDuration,
					p.slidingStart.Format(time.RFC822Z),
					wait.Round(time.Second)*1)

				p.slidingCount = 0

				// Sleep, but wake immediately if the campaign is paused or
				// cancelled. Without this, a pause issued mid-sleep leaves
				// the goroutine blocked for up to SlidingWindowDuration; on
				// resume a new pipe gets installed and the stale goroutine's
				// later cleanup would race with it.
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
				case <-p.done:
					if !timer.Stop() {
						<-timer.C
					}
					return false, nil
				}
			}
		}
	}

	return true, nil
}

// OnError keeps track of the number of errors that occur while sending messages
// and pauses the campaign if the error threshold is met.
func (p *pipe) OnError() {
	if p.m.cfg.MaxSendErrors < 1 {
		return
	}

	// If the error threshold is met, pause the campaign.
	count := p.errors.Add(1)
	if int(count) < p.m.cfg.MaxSendErrors {
		return
	}

	p.Stop(true)
	p.m.log.Printf("error count exceeded %d. pausing campaign %s", p.m.cfg.MaxSendErrors, p.camp.Name)
}

// Stop "marks" a campaign as stopped. It doesn't actually stop the processing
// of messages. That happens when every queued message in the campaign is processed,
// marking .wg, the waitgroup counter as done. That triggers cleanup().
//
// Closing p.done also wakes any goroutine currently parked in a sliding-window
// sleep so it can exit and let cleanup() run.
func (p *pipe) Stop(withErrors bool) {
	if withErrors {
		p.withErrors.Store(true)
	}

	// Already stopped.
	if p.stopped.Load() {
		return
	}

	p.stopped.Store(true)
	p.stopOnce.Do(func() {
		close(p.done)
	})
}

// newMessage returns a campaign message while internally incrementing the
// number of messages in the pipe wait group so that the status of every
// message can be atomically tracked.
func (p *pipe) newMessage(s models.Subscriber) (CampaignMessage, error) {
	msg, err := p.m.NewCampaignMessage(p.camp, s)
	if err != nil {
		return msg, err
	}

	msg.pipe = p
	p.wg.Add(1)

	return msg, nil
}

// cleanup finishes the campaign and updates the campaign status in the DB
// and also triggers a notification to the admin. This only triggers once
// a pipe's wg counter is fully exhausted, draining all messages in its queue.
func (p *pipe) cleanup() {
	defer func() {
		p.m.pipesMut.Lock()
		// Pointer-identity check: only remove the map entry if it still
		// points at THIS pipe. If a successor pipe has already been
		// installed for this campaign (e.g. after a stale pause/resume
		// cycle), leave it in place — otherwise the campaign goes silent
		// until the container restarts. Friday May 8 stall traced to this.
		if cur, ok := p.m.pipes[p.camp.ID]; ok && cur == p {
			delete(p.m.pipes, p.camp.ID)
		}
		p.m.pipesMut.Unlock()
	}()

	// Update campaign's 'sent count.
	if err := p.m.store.UpdateCampaignCounts(p.camp.ID, 0, int(p.sent.Load()), int(p.lastID.Load())); err != nil {
		p.m.log.Printf("error updating campaign counts (%s): %v", p.camp.Name, err)
	}

	// The campaign was auto-paused due to errors.
	if p.withErrors.Load() {
		if err := p.m.store.UpdateCampaignStatus(p.camp.ID, models.CampaignStatusPaused); err != nil {
			p.m.log.Printf("error updating campaign (%s) status to %s: %v", p.camp.Name, models.CampaignStatusPaused, err)
		} else {
			p.m.log.Printf("set campaign (%s) to %s", p.camp.Name, models.CampaignStatusPaused)
		}

		_ = p.m.sendNotif(p.camp, models.CampaignStatusPaused, "Too many errors")
		return
	}

	// The campaign was manually stopped (pause, cancel).
	if p.stopped.Load() {
		p.m.log.Printf("stop processing campaign (%s)", p.camp.Name)
		return
	}

	// Campaign wasn't manually stopped and subscribers were naturally exhausted.
	// Fetch the up-to-date campaign status from the DB.
	c, err := p.m.store.GetCampaign(p.camp.ID)
	if err != nil {
		p.m.log.Printf("error fetching campaign (%s) for ending: %v", p.camp.Name, err)
		return
	}

	// Evergreen campaigns never transition to finished — they wait in the
	// running state for future subscribers to join the list. The evergreen
	// scanner goroutine (see manager.Run → scanEvergreenCampaigns) resets
	// last_subscriber_id on a periodic tick so freshly-added subscribers
	// get picked up on the next normal scan cycle.
	if c.IsEvergreen && c.Status == models.CampaignStatusRunning {
		p.m.log.Printf("evergreen campaign (%s) drained initial send; staying in running", p.camp.Name)
	} else if c.Status == models.CampaignStatusRunning || c.Status == models.CampaignStatusScheduled {
		// If a running campaign has exhausted subscribers, it's finished.
		c.Status = models.CampaignStatusFinished
		if err := p.m.store.UpdateCampaignStatus(p.camp.ID, models.CampaignStatusFinished); err != nil {
			p.m.log.Printf("error finishing campaign (%s): %v", p.camp.Name, err)
		} else {
			p.m.log.Printf("campaign (%s) finished", p.camp.Name)
		}
	} else {
		p.m.log.Printf("finish processing campaign (%s)", p.camp.Name)
	}

	// Notify admin.
	_ = p.m.sendNotif(c, c.Status, "")
}
