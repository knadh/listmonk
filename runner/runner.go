package runner

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/knadh/listmonk/messenger"
	"github.com/knadh/listmonk/models"
)

const (
	batchSize = 10000

	// BaseTPL is the name of the base template.
	BaseTPL = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"
)

// DataSource represents a data backend, such as a database,
// that provides subscriber and campaign records.
type DataSource interface {
	NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error)
	NextSubscribers(campID, limit int) ([]*models.Subscriber, error)
	GetCampaign(campID int) (*models.Campaign, error)
	UpdateCampaignStatus(campID int, status string) error
	CreateLink(url string) (string, error)
}

// Runner handles the scheduling, processing, and queuing of campaigns
// and message pushes.
type Runner struct {
	cfg        Config
	src        DataSource
	messengers map[string]messenger.Messenger
	notifCB    models.AdminNotifCallback
	logger     *log.Logger

	// Campaigns that are currently running.
	camps map[int]*models.Campaign

	// Links generated using Track() are cached here so as to not query
	// the database for the link UUID for every message sent. This has to
	// be locked as it may be used externally when previewing campaigns.
	links      map[string]string
	linksMutex sync.RWMutex

	subFetchQueue  chan *models.Campaign
	msgQueue       chan *Message
	msgErrorQueue  chan msgError
	msgErrorCounts map[int]int
}

// Message represents an active subscriber that's being processed.
type Message struct {
	Campaign       *models.Campaign
	Subscriber     *models.Subscriber
	UnsubscribeURL string
	Body           []byte
	from           string
	to             string
}

// Config has parameters for configuring the runner.
type Config struct {
	Concurrency    int
	MaxSendErrors  int
	RequeueOnError bool
	FromEmail      string
	LinkTrackURL   string
	UnsubscribeURL string
	ViewTrackURL   string
}

type msgError struct {
	camp *models.Campaign
	err  error
}

// New returns a new instance of Mailer.
func New(cfg Config, src DataSource, notifCB models.AdminNotifCallback, l *log.Logger) *Runner {
	r := Runner{
		cfg:            cfg,
		src:            src,
		notifCB:        notifCB,
		logger:         l,
		messengers:     make(map[string]messenger.Messenger),
		camps:          make(map[int]*models.Campaign, 0),
		links:          make(map[string]string, 0),
		subFetchQueue:  make(chan *models.Campaign, cfg.Concurrency),
		msgQueue:       make(chan *Message, cfg.Concurrency),
		msgErrorQueue:  make(chan msgError, cfg.MaxSendErrors),
		msgErrorCounts: make(map[int]int),
	}

	return &r
}

// NewMessage creates and returns a Message that is made available
// to message templates while they're compiled.
func (r *Runner) NewMessage(c *models.Campaign, s *models.Subscriber) *Message {
	return &Message{
		from:           c.FromEmail,
		to:             s.Email,
		Campaign:       c,
		Subscriber:     s,
		UnsubscribeURL: fmt.Sprintf(r.cfg.UnsubscribeURL, c.UUID, s.UUID),
	}
}

// AddMessenger adds a Messenger messaging backend to the runner process.
func (r *Runner) AddMessenger(msg messenger.Messenger) error {
	id := msg.Name()
	if _, ok := r.messengers[id]; ok {
		return fmt.Errorf("messenger '%s' is already loaded", id)
	}
	r.messengers[id] = msg

	return nil
}

// GetMessengerNames returns the list of registered messengers.
func (r *Runner) GetMessengerNames() []string {
	var names []string
	for n := range r.messengers {
		names = append(names, n)
	}

	return names
}

// HasMessenger checks if a given messenger is registered.
func (r *Runner) HasMessenger(id string) bool {
	_, ok := r.messengers[id]
	return ok
}

// Run is a blocking function (and hence should be invoked as a goroutine)
// that scans the source db at regular intervals for pending campaigns,
// and queues them for processing. The process queue fetches batches of
// subscribers and pushes messages to them for each queued campaign
// until all subscribers are exhausted, at which point, a campaign is marked
// as "finished".
func (r *Runner) Run(tick time.Duration) {
	go func() {
		t := time.NewTicker(tick)
		for {
			select {
			// Periodically scan the data source for campaigns to process.
			case <-t.C:
				campaigns, err := r.src.NextCampaigns(r.getPendingCampaignIDs())
				if err != nil {
					r.logger.Printf("error fetching campaigns: %v", err)
					continue
				}

				for _, c := range campaigns {
					if err := r.addCampaign(c); err != nil {
						r.logger.Printf("error processing campaign (%s): %v", c.Name, err)
						continue
					}
					r.logger.Printf("start processing campaign (%s)", c.Name)

					// If subscriber processing is busy, move on. Blocking and waiting
					// can end up in a race condition where the waiting campaign's
					// state in the data source has changed.
					select {
					case r.subFetchQueue <- c:
					default:
					}
				}

				// Aggregate errors from sending messages to check against the error threshold
				// after which a campaign is paused.
			case e := <-r.msgErrorQueue:
				if r.cfg.MaxSendErrors < 1 {
					continue
				}

				// If the error threshold is met, pause the campaign.
				r.msgErrorCounts[e.camp.ID]++
				if r.msgErrorCounts[e.camp.ID] >= r.cfg.MaxSendErrors {
					r.logger.Printf("error counted exceeded %d. pausing campaign %s",
						r.cfg.MaxSendErrors, e.camp.Name)

					if r.isCampaignProcessing(e.camp.ID) {
						r.exhaustCampaign(e.camp, models.CampaignStatusPaused)
					}
					delete(r.msgErrorCounts, e.camp.ID)

					// Notify admins.
					r.sendNotif(e.camp,
						models.CampaignStatusPaused,
						"Too many errors")
				}
			}
		}
	}()

	// Fetch the next set of subscribers for a campaign and process them.
	for c := range r.subFetchQueue {
		has, err := r.nextSubscribers(c, batchSize)
		if err != nil {
			r.logger.Printf("error processing campaign batch (%s): %v", c.Name, err)
			continue
		}

		if has {
			// There are more subscribers to fetch.
			r.subFetchQueue <- c
		} else if r.isCampaignProcessing(c.ID) {
			// There are no more subscribers. Either the campaign status
			// has changed or all subscribers have been processed.
			newC, err := r.exhaustCampaign(c, "")
			if err != nil {
				r.logger.Printf("error exhausting campaign (%s): %v", c.Name, err)
				continue
			}
			r.sendNotif(newC, newC.Status, "")
		}
	}

}

// SpawnWorkers spawns workers goroutines that push out messages.
func (r *Runner) SpawnWorkers() {
	for i := 0; i < r.cfg.Concurrency; i++ {
		go func() {
			for m := range r.msgQueue {
				if !r.isCampaignProcessing(m.Campaign.ID) {
					continue
				}

				err := r.messengers[m.Campaign.MessengerID].Push(
					m.from,
					[]string{m.to},
					m.Campaign.Subject,
					m.Body)
				if err != nil {
					r.logger.Printf("error sending message in campaign %s: %v",
						m.Campaign.Name, err)

					select {
					case r.msgErrorQueue <- msgError{camp: m.Campaign, err: err}:
					default:
					}
				}
			}
		}()
	}
}

// addCampaign adds a campaign to the process queue.
func (r *Runner) addCampaign(c *models.Campaign) error {
	// Validate messenger.
	if _, ok := r.messengers[c.MessengerID]; !ok {
		r.src.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return fmt.Errorf("unknown messenger %s on campaign %s", c.MessengerID, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(r.TemplateFuncs(c)); err != nil {
		return err
	}

	// Add the campaign to the active map.
	r.camps[c.ID] = c
	return nil
}

// getPendingCampaignIDs returns the IDs of campaigns currently being processed.
func (r *Runner) getPendingCampaignIDs() []int64 {
	// Needs to return an empty slice in case there are no campaigns.
	ids := make([]int64, 0)
	for _, c := range r.camps {
		ids = append(ids, int64(c.ID))
	}

	return ids
}

// nextSubscribers processes the next batch of subscribers in a given campaign.
// If returns a bool indicating whether there any subscribers were processed
// in the current batch or not. This can happen when all the subscribers
// have been processed, or if a campaign has been paused or cancelled abruptly.
func (r *Runner) nextSubscribers(c *models.Campaign, batchSize int) (bool, error) {
	// Fetch a batch of subscribers.
	subs, err := r.src.NextSubscribers(c.ID, batchSize)
	if err != nil {
		return false, fmt.Errorf("error fetching campaign subscribers (%s): %v", c.Name, err)
	}

	// There are no subscribers.
	if len(subs) == 0 {
		return false, nil
	}

	// Push messages.
	for _, s := range subs {
		m := r.NewMessage(c, s)
		if err := m.Render(); err != nil {
			r.logger.Printf("error rendering message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		r.msgQueue <- m
	}

	return true, nil
}

// isCampaignProcessing checks if the campaign is bing processed.
func (r *Runner) isCampaignProcessing(id int) bool {
	_, ok := r.camps[id]
	return ok
}

func (r *Runner) exhaustCampaign(c *models.Campaign, status string) (*models.Campaign, error) {
	delete(r.camps, c.ID)

	// A status has been passed. Change the campaign's status
	// without further checks.
	if status != "" {
		if err := r.src.UpdateCampaignStatus(c.ID, status); err != nil {
			r.logger.Printf("error updating campaign (%s) status to %s: %v", c.Name, status, err)
		} else {
			r.logger.Printf("set campaign (%s) to %s", c.Name, status)
		}
		return c, nil
	}

	// Fetch the up-to-date campaign status from the source.
	cm, err := r.src.GetCampaign(c.ID)
	if err != nil {
		return nil, err
	}

	// If a running campaign has exhausted subscribers, it's finished.
	if cm.Status == models.CampaignStatusRunning {
		cm.Status = models.CampaignStatusFinished
		if err := r.src.UpdateCampaignStatus(c.ID, models.CampaignStatusFinished); err != nil {
			r.logger.Printf("error finishing campaign (%s): %v", c.Name, err)
		} else {
			r.logger.Printf("campaign (%s) finished", c.Name)
		}
	} else {
		r.logger.Printf("stop processing campaign (%s)", c.Name)
	}

	return cm, nil
}

// Render takes a Message, executes its pre-compiled Campaign.Tpl
// and applies the resultant bytes to Message.body to be used in messages.
func (m *Message) Render() error {
	out := bytes.Buffer{}
	if err := m.Campaign.Tpl.ExecuteTemplate(&out, models.BaseTpl, m); err != nil {
		return err
	}
	m.Body = out.Bytes()
	return nil
}

// trackLink register a URL and return its UUID to be used in message templates
// for tracking links.
func (r *Runner) trackLink(url, campUUID, subUUID string) string {
	r.linksMutex.RLock()
	if uu, ok := r.links[url]; ok {
		r.linksMutex.RUnlock()
		return fmt.Sprintf(r.cfg.LinkTrackURL, uu, campUUID, subUUID)
	}
	r.linksMutex.RUnlock()

	// Register link.
	uu, err := r.src.CreateLink(url)
	if err != nil {
		r.logger.Printf("error registering tracking for link '%s': %v", url, err)

		// If the registration fails, fail over to the original URL.
		return url
	}

	r.linksMutex.Lock()
	r.links[url] = uu
	r.linksMutex.Unlock()

	return fmt.Sprintf(r.cfg.LinkTrackURL, uu, campUUID, subUUID)
}

// sendNotif sends a notification to registered admin e-mails.
func (r *Runner) sendNotif(c *models.Campaign, status, reason string) error {
	var (
		subject = fmt.Sprintf("%s: %s", strings.Title(status), c.Name)
		data    = map[string]interface{}{
			"ID":     c.ID,
			"Name":   c.Name,
			"Status": status,
			"Sent":   c.Sent,
			"ToSend": c.ToSend,
			"Reason": reason,
		}
	)

	return r.notifCB(subject, data)
}

// TemplateFuncs returns the template functions to be applied into
// compiled campaign templates.
func (r *Runner) TemplateFuncs(c *models.Campaign) template.FuncMap {
	return template.FuncMap{
		"TrackLink": func(url, campUUID, subUUID string) string {
			return r.trackLink(url, campUUID, subUUID)
		},
		"TrackView": func(campUUID, subUUID string) template.HTML {
			return template.HTML(fmt.Sprintf(`<img src="%s" alt="" />`,
				fmt.Sprintf(r.cfg.ViewTrackURL, campUUID, subUUID)))
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
	}
}
