package manager

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

// Manager handles the scheduling, processing, and queuing of campaigns
// and message pushes.
type Manager struct {
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
	Campaign   *models.Campaign
	Subscriber *models.Subscriber
	Body       []byte

	from string
	to   string
}

// Config has parameters for configuring the manager.
type Config struct {
	Concurrency    int
	MaxSendErrors  int
	RequeueOnError bool
	FromEmail      string
	LinkTrackURL   string
	UnsubURL       string
	OptinURL       string
	ViewTrackURL   string
}

type msgError struct {
	camp *models.Campaign
	err  error
}

// New returns a new instance of Mailer.
func New(cfg Config, src DataSource, notifCB models.AdminNotifCallback, l *log.Logger) *Manager {
	return &Manager{
		cfg:            cfg,
		src:            src,
		notifCB:        notifCB,
		logger:         l,
		messengers:     make(map[string]messenger.Messenger),
		camps:          make(map[int]*models.Campaign),
		links:          make(map[string]string),
		subFetchQueue:  make(chan *models.Campaign, cfg.Concurrency),
		msgQueue:       make(chan *Message, cfg.Concurrency),
		msgErrorQueue:  make(chan msgError, cfg.MaxSendErrors),
		msgErrorCounts: make(map[int]int),
	}
}

// NewMessage creates and returns a Message that is made available
// to message templates while they're compiled.
func (m *Manager) NewMessage(c *models.Campaign, s *models.Subscriber) *Message {
	return &Message{
		Campaign:   c,
		Subscriber: s,

		from: c.FromEmail,
		to:   s.Email,
	}
}

// AddMessenger adds a Messenger messaging backend to the manager.
func (m *Manager) AddMessenger(msg messenger.Messenger) error {
	id := msg.Name()
	if _, ok := m.messengers[id]; ok {
		return fmt.Errorf("messenger '%s' is already loaded", id)
	}
	m.messengers[id] = msg

	return nil
}

// GetMessengerNames returns the list of registered messengers.
func (m *Manager) GetMessengerNames() []string {
	names := make([]string, 0, len(m.messengers))
	for n := range m.messengers {
		names = append(names, n)
	}

	return names
}

// HasMessenger checks if a given messenger is registered.
func (m *Manager) HasMessenger(id string) bool {
	_, ok := m.messengers[id]
	return ok
}

// Run is a blocking function (and hence should be invoked as a goroutine)
// that scans the source db at regular intervals for pending campaigns,
// and queues them for processing. The process queue fetches batches of
// subscribers and pushes messages to them for each queued campaign
// until all subscribers are exhausted, at which point, a campaign is marked
// as "finished".
func (m *Manager) Run(tick time.Duration) {
	go func() {
		t := time.NewTicker(tick)
		for {
			select {
			// Periodically scan the data source for campaigns to process.
			case <-t.C:
				campaigns, err := m.src.NextCampaigns(m.getPendingCampaignIDs())
				if err != nil {
					m.logger.Printf("error fetching campaigns: %v", err)
					continue
				}

				for _, c := range campaigns {
					if err := m.addCampaign(c); err != nil {
						m.logger.Printf("error processing campaign (%s): %v", c.Name, err)
						continue
					}
					m.logger.Printf("start processing campaign (%s)", c.Name)

					// If subscriber processing is busy, move on. Blocking and waiting
					// can end up in a race condition where the waiting campaign's
					// state in the data source has changed.
					select {
					case m.subFetchQueue <- c:
					default:
					}
				}

				// Aggregate errors from sending messages to check against the error threshold
				// after which a campaign is paused.
			case e := <-m.msgErrorQueue:
				if m.cfg.MaxSendErrors < 1 {
					continue
				}

				// If the error threshold is met, pause the campaign.
				m.msgErrorCounts[e.camp.ID]++
				if m.msgErrorCounts[e.camp.ID] >= m.cfg.MaxSendErrors {
					m.logger.Printf("error counted exceeded %d. pausing campaign %s",
						m.cfg.MaxSendErrors, e.camp.Name)

					if m.isCampaignProcessing(e.camp.ID) {
						m.exhaustCampaign(e.camp, models.CampaignStatusPaused)
					}
					delete(m.msgErrorCounts, e.camp.ID)

					// Notify admins.
					m.sendNotif(e.camp,
						models.CampaignStatusPaused,
						"Too many errors")
				}
			}
		}
	}()

	// Fetch the next set of subscribers for a campaign and process them.
	for c := range m.subFetchQueue {
		has, err := m.nextSubscribers(c, batchSize)
		if err != nil {
			m.logger.Printf("error processing campaign batch (%s): %v", c.Name, err)
			continue
		}

		if has {
			// There are more subscribers to fetch.
			m.subFetchQueue <- c
		} else if m.isCampaignProcessing(c.ID) {
			// There are no more subscribers. Either the campaign status
			// has changed or all subscribers have been processed.
			newC, err := m.exhaustCampaign(c, "")
			if err != nil {
				m.logger.Printf("error exhausting campaign (%s): %v", c.Name, err)
				continue
			}
			m.sendNotif(newC, newC.Status, "")
		}
	}
}

// SpawnWorkers spawns workers goroutines that push out messages.
func (m *Manager) SpawnWorkers() {
	for i := 0; i < m.cfg.Concurrency; i++ {
		go func() {
			for msg := range m.msgQueue {
				if !m.isCampaignProcessing(msg.Campaign.ID) {
					continue
				}

				err := m.messengers[msg.Campaign.MessengerID].Push(
					msg.from,
					[]string{msg.to},
					msg.Campaign.Subject,
					msg.Body, nil)
				if err != nil {
					m.logger.Printf("error sending message in campaign %s: %v",
						msg.Campaign.Name, err)

					select {
					case m.msgErrorQueue <- msgError{camp: msg.Campaign, err: err}:
					default:
					}
				}
			}
		}()
	}
}

// addCampaign adds a campaign to the process queue.
func (m *Manager) addCampaign(c *models.Campaign) error {
	// Validate messenger.
	if _, ok := m.messengers[c.MessengerID]; !ok {
		m.src.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return fmt.Errorf("unknown messenger %s on campaign %s", c.MessengerID, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		return err
	}

	// Add the campaign to the active map.
	m.camps[c.ID] = c
	return nil
}

// getPendingCampaignIDs returns the IDs of campaigns currently being processed.
func (m *Manager) getPendingCampaignIDs() []int64 {
	// Needs to return an empty slice in case there are no campaigns.
	ids := make([]int64, 0)
	for _, c := range m.camps {
		ids = append(ids, int64(c.ID))
	}

	return ids
}

// nextSubscribers processes the next batch of subscribers in a given campaign.
// If returns a bool indicating whether there any subscribers were processed
// in the current batch or not. This can happen when all the subscribers
// have been processed, or if a campaign has been paused or cancelled abruptly.
func (m *Manager) nextSubscribers(c *models.Campaign, batchSize int) (bool, error) {
	// Fetch a batch of subscribers.
	subs, err := m.src.NextSubscribers(c.ID, batchSize)
	if err != nil {
		return false, fmt.Errorf("error fetching campaign subscribers (%s): %v", c.Name, err)
	}

	// There are no subscribers.
	if len(subs) == 0 {
		return false, nil
	}

	// Push messages.
	for _, s := range subs {
		msg := m.NewMessage(c, s)
		if err := msg.Render(); err != nil {
			m.logger.Printf("error rendering message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		m.msgQueue <- msg
	}

	return true, nil
}

// isCampaignProcessing checks if the campaign is bing processed.
func (m *Manager) isCampaignProcessing(id int) bool {
	_, ok := m.camps[id]
	return ok
}

func (m *Manager) exhaustCampaign(c *models.Campaign, status string) (*models.Campaign, error) {
	delete(m.camps, c.ID)

	// A status has been passed. Change the campaign's status
	// without further checks.
	if status != "" {
		if err := m.src.UpdateCampaignStatus(c.ID, status); err != nil {
			m.logger.Printf("error updating campaign (%s) status to %s: %v", c.Name, status, err)
		} else {
			m.logger.Printf("set campaign (%s) to %s", c.Name, status)
		}
		return c, nil
	}

	// Fetch the up-to-date campaign status from the source.
	cm, err := m.src.GetCampaign(c.ID)
	if err != nil {
		return nil, err
	}

	// If a running campaign has exhausted subscribers, it's finished.
	if cm.Status == models.CampaignStatusRunning {
		cm.Status = models.CampaignStatusFinished
		if err := m.src.UpdateCampaignStatus(c.ID, models.CampaignStatusFinished); err != nil {
			m.logger.Printf("error finishing campaign (%s): %v", c.Name, err)
		} else {
			m.logger.Printf("campaign (%s) finished", c.Name)
		}
	} else {
		m.logger.Printf("stop processing campaign (%s)", c.Name)
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
func (m *Manager) trackLink(url, campUUID, subUUID string) string {
	m.linksMutex.RLock()
	if uu, ok := m.links[url]; ok {
		m.linksMutex.RUnlock()
		return fmt.Sprintf(m.cfg.LinkTrackURL, uu, campUUID, subUUID)
	}
	m.linksMutex.RUnlock()

	// Register link.
	uu, err := m.src.CreateLink(url)
	if err != nil {
		m.logger.Printf("error registering tracking for link '%s': %v", url, err)

		// If the registration fails, fail over to the original URL.
		return url
	}

	m.linksMutex.Lock()
	m.links[url] = uu
	m.linksMutex.Unlock()

	return fmt.Sprintf(m.cfg.LinkTrackURL, uu, campUUID, subUUID)
}

// sendNotif sends a notification to registered admin e-mails.
func (m *Manager) sendNotif(c *models.Campaign, status, reason string) error {
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

	return m.notifCB(subject, data)
}

// TemplateFuncs returns the template functions to be applied into
// compiled campaign templates.
func (m *Manager) TemplateFuncs(c *models.Campaign) template.FuncMap {
	return template.FuncMap{
		"TrackLink": func(url string, msg *Message) string {
			return m.trackLink(url, msg.Campaign.UUID, msg.Subscriber.UUID)
		},
		"TrackView": func(msg *Message) template.HTML {
			return template.HTML(fmt.Sprintf(`<img src="%s" alt="" />`,
				fmt.Sprintf(m.cfg.ViewTrackURL, msg.Campaign.UUID, msg.Subscriber.UUID)))
		},
		"UnsubscribeURL": func(msg *Message) string {
			return fmt.Sprintf(m.cfg.UnsubURL, c.UUID, msg.Subscriber.UUID)
		},
		"OptinURL": func(msg *Message) string {
			// Add list IDs.
			// TODO: Show private lists list on optin e-mail
			return fmt.Sprintf(m.cfg.OptinURL, msg.Subscriber.UUID, "")
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
	}
}
