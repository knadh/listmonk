package runner

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
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
	PauseCampaign(campID int) error
	CancelCampaign(campID int) error
	FinishCampaign(campID int) error
}

// Runner handles the scheduling, processing, and queuing of campaigns
// and message pushes.
type Runner struct {
	cfg        Config
	src        DataSource
	messengers map[string]messenger.Messenger
	logger     *log.Logger

	// Campaigns that are currently running.
	camps map[int]*models.Campaign

	msgQueue      chan Message
	subFetchQueue chan *models.Campaign
}

// Message represents an active subscriber that's being processed.
type Message struct {
	Campaign       *models.Campaign
	Subscriber     *models.Subscriber
	UnsubscribeURL string

	body []byte
	to   string
}

// Config has parameters for configuring the runner.
type Config struct {
	Concurrency    int
	UnsubscribeURL string
}

// New returns a new instance of Mailer.
func New(cfg Config, src DataSource, l *log.Logger) *Runner {
	r := Runner{
		cfg:           cfg,
		messengers:    make(map[string]messenger.Messenger),
		src:           src,
		camps:         make(map[int]*models.Campaign, 0),
		logger:        l,
		subFetchQueue: make(chan *models.Campaign, 100),
		msgQueue:      make(chan Message, cfg.Concurrency),
	}

	return &r
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
	var (
		tScanCampaigns = time.NewTicker(tick)
	)

	for {
		select {
		// Fetch all 'running campaigns that aren't being processed.
		case <-tScanCampaigns.C:
			campaigns, err := r.src.NextCampaigns(r.getPendingCampaignIDs())
			if err != nil {
				r.logger.Printf("error fetching campaigns: %v", err)
				return
			}

			for _, c := range campaigns {
				if err := r.addCampaign(c); err != nil {
					r.logger.Printf("error processing campaign (%s): %v", c.Name, err)
					continue
				}

				r.logger.Printf("start processing campaign (%s)", c.Name)
				r.subFetchQueue <- c
			}

			// Fetch next set of subscribers for the incoming campaign ID
			// and process them.
		case c := <-r.subFetchQueue:
			has, err := r.nextSubscribers(c, batchSize)
			if err != nil {
				r.logger.Printf("error processing campaign batch (%s): %v", c.Name, err)
			}

			if has {
				// There are more subscribers to fetch.
				r.subFetchQueue <- c
			} else {
				// No subscribers.
				if err := r.processExhaustedCampaign(c); err != nil {
					r.logger.Printf("error processing campaign (%s): %v", c.Name, err)
				}
			}
		}
	}
}

// SpawnWorkers spawns workers goroutines that push out messages.
func (r *Runner) SpawnWorkers() {
	for i := 0; i < r.cfg.Concurrency; i++ {
		go func(ch chan Message) {
			for {
				select {
				case m := <-ch:
					r.messengers[m.Campaign.MessengerID].Push(
						m.Campaign.FromEmail,
						m.Subscriber.Email,
						m.Campaign.Subject,
						m.body)
				}
			}
		}(r.msgQueue)
	}
}

// addCampaign adds a campaign to the process queue.
func (r *Runner) addCampaign(c *models.Campaign) error {
	var tplErr error

	c.Tpl, tplErr = CompileMessageTemplate(c.TemplateBody, c.Body)
	if tplErr != nil {
		return tplErr
	}

	// Validate messenger.
	if _, ok := r.messengers[c.MessengerID]; !ok {
		r.src.CancelCampaign(c.ID)
		return fmt.Errorf("unknown messenger %s on campaign %s", c.MessengerID, c.Name)
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
		to, body, err := r.makeMessage(c, s)
		if err != nil {
			r.logger.Printf("error preparing message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Send the message.
		r.msgQueue <- Message{Campaign: c,
			Subscriber: s,
			to:         to,
			body:       body}
	}

	return true, nil
}

func (r *Runner) processExhaustedCampaign(c *models.Campaign) error {
	cm, err := r.src.GetCampaign(c.ID)
	if err != nil {
		return err
	}

	// If a running campaign has exhausted subscribers, it's finished.
	// Otherwise, it's paused or cancelled.
	if cm.Status == models.CampaignStatusRunning {
		if err := r.src.FinishCampaign(c.ID); err != nil {
			r.logger.Printf("error finishing campaign (%s): %v", c.Name, err)
		} else {
			r.logger.Printf("campaign (%s) finished", c.Name)
		}
	} else {
		r.logger.Printf("stop processing campaign (%s)", c.Name)
	}

	delete(r.camps, c.ID)
	return nil
}

// makeMessage prepares a campaign message for a subscriber and returns
// the 'to' address and the body.
func (r *Runner) makeMessage(c *models.Campaign, s *models.Subscriber) (string, []byte, error) {
	// Render the message body.
	var (
		out    = bytes.Buffer{}
		tplMsg = Message{Campaign: c,
			Subscriber:     s,
			UnsubscribeURL: fmt.Sprintf(r.cfg.UnsubscribeURL, c.UUID, s.UUID)}
	)
	if err := c.Tpl.ExecuteTemplate(&out, BaseTPL, tplMsg); err != nil {
		return "", nil, err
	}

	return s.Email, out.Bytes(), nil
}

// CompileMessageTemplate takes a base template body string and a child (message) template
// body string, compiles both and inserts the child template as the named template "content"
// and returns the resultant template.
func CompileMessageTemplate(baseBody, childBody string) (*template.Template, error) {
	// Compile the base template.
	baseTPL, err := template.New(BaseTPL).Parse(baseBody)
	if err != nil {
		return nil, fmt.Errorf("error compiling base template: %v", err)
	}

	// Compile the campaign message.
	msgTpl, err := template.New(ContentTpl).Parse(childBody)
	if err != nil {
		return nil, fmt.Errorf("error compiling message: %v", err)
	}

	out, err := baseTPL.AddParseTree(ContentTpl, msgTpl.Tree)
	if err != nil {
		return nil, fmt.Errorf("error inserting child template: %v", err)
	}

	return out, nil
}
