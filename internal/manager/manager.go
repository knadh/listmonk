package manager

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/models"
)

const (
	// BaseTPL is the name of the base template.
	BaseTPL = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"
)

// DataSource represents a data backend, such as a database,
// that provides subscriber and campaign records.
type DataSource interface {
	NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error)
	NextSubscribers(campID, limit int) ([]models.Subscriber, error)
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
	camps      map[int]*models.Campaign
	campsMutex sync.RWMutex

	// Links generated using Track() are cached here so as to not query
	// the database for the link UUID for every message sent. This has to
	// be locked as it may be used externally when previewing campaigns.
	links      map[string]string
	linksMutex sync.RWMutex

	subFetchQueue      chan *models.Campaign
	campMsgQueue       chan CampaignMessage
	campMsgErrorQueue  chan msgError
	campMsgErrorCounts map[int]int
	msgQueue           chan Message
}

// CampaignMessage represents an instance of campaign message to be pushed out,
// specific to a subscriber, via the campaign's messenger.
type CampaignMessage struct {
	Campaign   *models.Campaign
	Subscriber models.Subscriber

	from     string
	to       string
	subject  string
	body     []byte
	unsubURL string
}

// Message represents a generic message to be pushed to a messenger.
type Message struct {
	From      string
	To        []string
	Subject   string
	Body      []byte
	Messenger string
}

// Config has parameters for configuring the manager.
type Config struct {
	// Number of subscribers to pull from the DB in a single iteration.
	BatchSize int

	Concurrency    int
	MessageRate    int
	MaxSendErrors  int
	RequeueOnError bool
	FromEmail      string
	LinkTrackURL   string
	UnsubURL       string
	OptinURL       string
	MessageURL     string
	ViewTrackURL   string
}

type msgError struct {
	camp *models.Campaign
	err  error
}

// New returns a new instance of Mailer.
func New(cfg Config, src DataSource, notifCB models.AdminNotifCallback, l *log.Logger) *Manager {
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 1000
	}
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.MessageRate < 1 {
		cfg.MessageRate = 1
	}

	return &Manager{
		cfg:                cfg,
		src:                src,
		notifCB:            notifCB,
		logger:             l,
		messengers:         make(map[string]messenger.Messenger),
		camps:              make(map[int]*models.Campaign),
		links:              make(map[string]string),
		subFetchQueue:      make(chan *models.Campaign, cfg.Concurrency),
		campMsgQueue:       make(chan CampaignMessage, cfg.Concurrency*2),
		msgQueue:           make(chan Message, cfg.Concurrency),
		campMsgErrorQueue:  make(chan msgError, cfg.MaxSendErrors),
		campMsgErrorCounts: make(map[int]int),
	}
}

// NewCampaignMessage creates and returns a CampaignMessage that is made available
// to message templates while they're compiled. It represents a message from
// a campaign that's bound to a single Subscriber.
func (m *Manager) NewCampaignMessage(c *models.Campaign, s models.Subscriber) CampaignMessage {
	return CampaignMessage{
		Campaign:   c,
		Subscriber: s,

		subject:  c.Subject,
		from:     c.FromEmail,
		to:       s.Email,
		unsubURL: fmt.Sprintf(m.cfg.UnsubURL, c.UUID, s.UUID),
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

// PushMessage pushes a Message to be sent out by the workers.
func (m *Manager) PushMessage(msg Message) error {
	select {
	case m.msgQueue <- msg:
	case <-time.After(time.Second * 3):
		m.logger.Println("message push timed out: %'s'", msg.Subject)
		return errors.New("message push timed out")
	}
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

// Run is a blocking function (that should be invoked as a goroutine)
// that scans the data source at regular intervals for pending campaigns,
// and queues them for processing. The process queue fetches batches of
// subscribers and pushes messages to them for each queued campaign
// until all subscribers are exhausted, at which point, a campaign is marked
// as "finished".
func (m *Manager) Run(tick time.Duration) {
	go m.scanCampaigns(tick)

	// Spawn N message workers.
	for i := 0; i < m.cfg.Concurrency; i++ {
		go m.messageWorker()
	}

	// Fetch the next set of subscribers for a campaign and process them.
	for c := range m.subFetchQueue {
		has, err := m.nextSubscribers(c, m.cfg.BatchSize)
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

// messageWorker is a blocking function that listens to the message queue
// and pushes out incoming messages on it to the messenger.
func (m *Manager) messageWorker() {
	// Counter to keep track of the message / sec rate limit.
	numMsg := 0
	for {
		select {
		// Campaign message.
		case msg := <-m.campMsgQueue:
			// Pause on hitting the message rate.
			if numMsg >= m.cfg.MessageRate {
				time.Sleep(time.Second)
				numMsg = 0
			}
			numMsg++

			err := m.messengers[msg.Campaign.MessengerID].Push(
				msg.from, []string{msg.to}, msg.subject, msg.body, nil)
			if err != nil {
				m.logger.Printf("error sending message in campaign %s: %v", msg.Campaign.Name, err)

				select {
				case m.campMsgErrorQueue <- msgError{camp: msg.Campaign, err: err}:
				default:
				}
			}

		// Arbitrary message.
		case msg := <-m.msgQueue:
			err := m.messengers[msg.Messenger].Push(
				msg.From, msg.To, msg.Subject, msg.Body, nil)
			if err != nil {
				m.logger.Printf("error sending message '%s': %v", msg.Subject, err)
			}
		}
	}
}

// TemplateFuncs returns the template functions to be applied into
// compiled campaign templates.
func (m *Manager) TemplateFuncs(c *models.Campaign) template.FuncMap {
	return template.FuncMap{
		"TrackLink": func(url string, msg *CampaignMessage) string {
			return m.trackLink(url, msg.Campaign.UUID, msg.Subscriber.UUID)
		},
		"TrackView": func(msg *CampaignMessage) template.HTML {
			return template.HTML(fmt.Sprintf(`<img src="%s" alt="" />`,
				fmt.Sprintf(m.cfg.ViewTrackURL, msg.Campaign.UUID, msg.Subscriber.UUID)))
		},
		"UnsubscribeURL": func(msg *CampaignMessage) string {
			return msg.unsubURL
		},
		"OptinURL": func(msg *CampaignMessage) string {
			// Add list IDs.
			// TODO: Show private lists list on optin e-mail
			return fmt.Sprintf(m.cfg.OptinURL, msg.Subscriber.UUID, "")
		},
		"MessageURL": func(msg *CampaignMessage) string {
			return fmt.Sprintf(m.cfg.MessageURL, c.UUID, msg.Subscriber.UUID)
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
	}
}

// scanCampaigns is a blocking function that periodically scans the data source
// for campaigns to process and dispatches them to the manager.
func (m *Manager) scanCampaigns(tick time.Duration) {
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
		case e := <-m.campMsgErrorQueue:
			if m.cfg.MaxSendErrors < 1 {
				continue
			}

			// If the error threshold is met, pause the campaign.
			m.campMsgErrorCounts[e.camp.ID]++
			if m.campMsgErrorCounts[e.camp.ID] >= m.cfg.MaxSendErrors {
				m.logger.Printf("error counted exceeded %d. pausing campaign %s",
					m.cfg.MaxSendErrors, e.camp.Name)

				if m.isCampaignProcessing(e.camp.ID) {
					m.exhaustCampaign(e.camp, models.CampaignStatusPaused)
				}
				delete(m.campMsgErrorCounts, e.camp.ID)

				// Notify admins.
				m.sendNotif(e.camp, models.CampaignStatusPaused, "Too many errors")
			}
		}
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
	m.campsMutex.Lock()
	m.camps[c.ID] = c
	m.campsMutex.Unlock()
	return nil
}

// getPendingCampaignIDs returns the IDs of campaigns currently being processed.
func (m *Manager) getPendingCampaignIDs() []int64 {
	// Needs to return an empty slice in case there are no campaigns.
	m.campsMutex.RLock()
	ids := make([]int64, 0, len(m.camps))
	for _, c := range m.camps {
		ids = append(ids, int64(c.ID))
	}
	m.campsMutex.RUnlock()
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
		msg := m.NewCampaignMessage(c, s)
		if err := msg.Render(); err != nil {
			m.logger.Printf("error rendering message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		m.campMsgQueue <- msg
	}

	return true, nil
}

// isCampaignProcessing checks if the campaign is bing processed.
func (m *Manager) isCampaignProcessing(id int) bool {
	m.campsMutex.RLock()
	_, ok := m.camps[id]
	m.campsMutex.RUnlock()
	return ok
}

func (m *Manager) exhaustCampaign(c *models.Campaign, status string) (*models.Campaign, error) {
	m.campsMutex.Lock()
	delete(m.camps, c.ID)
	m.campsMutex.Unlock()

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

// Render takes a Message, executes its pre-compiled Campaign.Tpl
// and applies the resultant bytes to Message.body to be used in messages.
func (m *CampaignMessage) Render() error {
	out := bytes.Buffer{}

	// Render the subject if it's a template.
	if m.Campaign.SubjectTpl != nil {
		if err := m.Campaign.SubjectTpl.ExecuteTemplate(&out, models.ContentTpl, m); err != nil {
			return err
		}
		m.subject = out.String()
		out.Reset()
	}

	if err := m.Campaign.Tpl.ExecuteTemplate(&out, models.BaseTpl, m); err != nil {
		return err
	}
	m.body = out.Bytes()
	return nil
}

// Subject returns a copy of the message subject
func (m *CampaignMessage) Subject() string {
	return m.subject
}

// Body returns a copy of the message body.
func (m *CampaignMessage) Body() []byte {
	out := make([]byte, len(m.body))
	copy(out, m.body)
	return out
}
