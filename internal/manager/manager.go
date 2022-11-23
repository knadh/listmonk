package manager

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/models"
	"github.com/paulbellamy/ratecounter"
)

const (
	// BaseTPL is the name of the base template.
	BaseTPL = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"

	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// Store represents a data backend, such as a database,
// that provides subscriber and campaign records.
type Store interface {
	NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error)
	NextSubscribers(campID, limit int) ([]models.Subscriber, error)
	GetCampaign(campID int) (*models.Campaign, error)
	UpdateCampaignStatus(campID int, status string) error
	CreateLink(url string) (string, error)
	BlocklistSubscriber(id int64) error
	DeleteSubscriber(id int64) error
}

// CampStats contains campaign stats like per minute send rate.
type CampStats struct {
	SendRate int
}

// Manager handles the scheduling, processing, and queuing of campaigns
// and message pushes.
type Manager struct {
	cfg        Config
	store      Store
	i18n       *i18n.I18n
	messengers map[string]messenger.Messenger
	notifCB    models.AdminNotifCallback
	logger     *log.Logger

	// Campaigns that are currently running.
	camps     map[int]*models.Campaign
	campRates map[int]*ratecounter.RateCounter
	campsMut  sync.RWMutex

	tpls    map[int]*models.Template
	tplsMut sync.RWMutex

	// Links generated using Track() are cached here so as to not query
	// the database for the link UUID for every message sent. This has to
	// be locked as it may be used externally when previewing campaigns.
	links    map[string]string
	linksMut sync.RWMutex

	subFetchQueue      chan *models.Campaign
	campMsgQueue       chan CampaignMessage
	campMsgErrorQueue  chan msgError
	campMsgErrorCounts map[int]int
	msgQueue           chan Message

	// Sliding window keeps track of the total number of messages sent in a period
	// and on reaching the specified limit, waits until the window is over before
	// sending further messages.
	slidingWindowNumMsg int
	slidingWindowStart  time.Time

	tplFuncs template.FuncMap
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
	altBody  []byte
	unsubURL string
}

// Message represents a generic message to be pushed to a messenger.
type Message struct {
	messenger.Message
	Subscriber models.Subscriber

	// Messenger is the messenger backend to use: email|postback.
	Messenger string
}

// Config has parameters for configuring the manager.
type Config struct {
	// Number of subscribers to pull from the DB in a single iteration.
	BatchSize             int
	Concurrency           int
	MessageRate           int
	MaxSendErrors         int
	SlidingWindow         bool
	SlidingWindowDuration time.Duration
	SlidingWindowRate     int
	RequeueOnError        bool
	FromEmail             string
	IndividualTracking    bool
	LinkTrackURL          string
	UnsubURL              string
	OptinURL              string
	MessageURL            string
	ViewTrackURL          string
	ArchiveURL            string
	UnsubHeader           bool

	// Interval to scan the DB for active campaign checkpoints.
	ScanInterval time.Duration

	// ScanCampaigns indicates whether this instance of manager will scan the DB
	// for active campaigns and process them.
	// This can be used to run multiple instances of listmonk
	// (exposed to the internet, private etc.) where only one does campaign
	// processing while the others handle other kinds of traffic.
	ScanCampaigns bool
}

type msgError struct {
	camp *models.Campaign
	err  error
}

var pushTimeout = time.Second * 3

// New returns a new instance of Mailer.
func New(cfg Config, store Store, notifCB models.AdminNotifCallback, i *i18n.I18n, l *log.Logger) *Manager {
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 1000
	}
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.MessageRate < 1 {
		cfg.MessageRate = 1
	}

	m := &Manager{
		cfg:                cfg,
		store:              store,
		i18n:               i,
		notifCB:            notifCB,
		logger:             l,
		messengers:         make(map[string]messenger.Messenger),
		camps:              make(map[int]*models.Campaign),
		campRates:          make(map[int]*ratecounter.RateCounter),
		tpls:               make(map[int]*models.Template),
		links:              make(map[string]string),
		subFetchQueue:      make(chan *models.Campaign, cfg.Concurrency),
		campMsgQueue:       make(chan CampaignMessage, cfg.Concurrency*2),
		msgQueue:           make(chan Message, cfg.Concurrency),
		campMsgErrorQueue:  make(chan msgError, cfg.MaxSendErrors),
		campMsgErrorCounts: make(map[int]int),
		slidingWindowStart: time.Now(),
	}
	m.tplFuncs = m.makeGnericFuncMap()

	return m
}

// NewCampaignMessage creates and returns a CampaignMessage that is made available
// to message templates while they're compiled. It represents a message from
// a campaign that's bound to a single Subscriber.
func (m *Manager) NewCampaignMessage(c *models.Campaign, s models.Subscriber) (CampaignMessage, error) {
	msg := CampaignMessage{
		Campaign:   c,
		Subscriber: s,

		subject:  c.Subject,
		from:     c.FromEmail,
		to:       s.Email,
		unsubURL: fmt.Sprintf(m.cfg.UnsubURL, c.UUID, s.UUID),
	}

	if err := msg.render(); err != nil {
		return msg, err
	}

	return msg, nil
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

// PushMessage pushes an arbitrary non-campaign Message to be sent out by the workers.
// It times out if the queue is busy.
func (m *Manager) PushMessage(msg Message) error {
	t := time.NewTicker(pushTimeout)
	defer t.Stop()

	select {
	case m.msgQueue <- msg:
	case <-t.C:
		m.logger.Printf("message push timed out: '%s'", msg.Subject)
		return errors.New("message push timed out")
	}
	return nil
}

// PushCampaignMessage pushes a campaign messages into a queue to be sent out by the workers.
// It times out if the queue is busy.
func (m *Manager) PushCampaignMessage(msg CampaignMessage) error {
	t := time.NewTicker(pushTimeout)
	defer t.Stop()

	select {
	case m.campMsgQueue <- msg:
	case <-t.C:
		m.logger.Printf("message push timed out: '%s'", msg.Subject())
		return errors.New("message push timed out")
	}
	return nil
}

// HasMessenger checks if a given messenger is registered.
func (m *Manager) HasMessenger(id string) bool {
	_, ok := m.messengers[id]
	return ok
}

// HasRunningCampaigns checks if there are any active campaigns.
func (m *Manager) HasRunningCampaigns() bool {
	m.campsMut.Lock()
	defer m.campsMut.Unlock()
	return len(m.camps) > 0
}

// GetCampaignStats returns campaign statistics.
func (m *Manager) GetCampaignStats(id int) CampStats {
	n := 0

	m.campsMut.Lock()
	if r, ok := m.campRates[id]; ok {
		n = int(r.Rate())
	}
	m.campsMut.Unlock()

	return CampStats{SendRate: n}
}

// Run is a blocking function (that should be invoked as a goroutine)
// that scans the data source at regular intervals for pending campaigns,
// and queues them for processing. The process queue fetches batches of
// subscribers and pushes messages to them for each queued campaign
// until all subscribers are exhausted, at which point, a campaign is marked
// as "finished".
func (m *Manager) Run() {
	if m.cfg.ScanCampaigns {
		go m.scanCampaigns(m.cfg.ScanInterval)
	}

	// Spawn N message workers.
	for i := 0; i < m.cfg.Concurrency; i++ {
		go m.worker()
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

// CacheTpl caches a template for ad-hoc use. This is currently only used by tx templates.
func (m *Manager) CacheTpl(id int, tpl *models.Template) {
	m.tplsMut.Lock()
	m.tpls[id] = tpl
	m.tplsMut.Unlock()
}

// DeleteTpl deletes a cached template.
func (m *Manager) DeleteTpl(id int) {
	m.tplsMut.Lock()
	delete(m.tpls, id)
	m.tplsMut.Unlock()
}

// GetTpl returns a cached template.
func (m *Manager) GetTpl(id int) (*models.Template, error) {
	m.tplsMut.RLock()
	tpl, ok := m.tpls[id]
	m.tplsMut.RUnlock()

	if !ok {
		return nil, fmt.Errorf("template %d not found", id)
	}

	return tpl, nil
}

// worker is a blocking function that perpetually listents to events (message) on different
// queues and processes them.
func (m *Manager) worker() {
	// Counter to keep track of the message / sec rate limit.
	numMsg := 0
	for {
		select {
		// Campaign message.
		case msg, ok := <-m.campMsgQueue:
			if !ok {
				return
			}

			// Pause on hitting the message rate.
			if numMsg >= m.cfg.MessageRate {
				time.Sleep(time.Second)
				numMsg = 0
			}
			numMsg++

			// Outgoing message.
			out := messenger.Message{
				From:        msg.from,
				To:          []string{msg.to},
				Subject:     msg.subject,
				ContentType: msg.Campaign.ContentType,
				Body:        msg.body,
				AltBody:     msg.altBody,
				Subscriber:  msg.Subscriber,
				Campaign:    msg.Campaign,
			}

			h := textproto.MIMEHeader{}
			h.Set(models.EmailHeaderCampaignUUID, msg.Campaign.UUID)
			h.Set(models.EmailHeaderSubscriberUUID, msg.Subscriber.UUID)

			// Attach List-Unsubscribe headers?
			if m.cfg.UnsubHeader {
				h.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
				h.Set("List-Unsubscribe", `<`+msg.unsubURL+`>`)
			}

			// Attach any custom headers.
			if len(msg.Campaign.Headers) > 0 {
				for _, set := range msg.Campaign.Headers {
					for hdr, val := range set {
						h.Add(hdr, val)
					}
				}
			}

			out.Headers = h

			if err := m.messengers[msg.Campaign.Messenger].Push(out); err != nil {
				m.logger.Printf("error sending message in campaign %s: subscriber %s: %v",
					msg.Campaign.Name, msg.Subscriber.UUID, err)

				select {
				case m.campMsgErrorQueue <- msgError{camp: msg.Campaign, err: err}:
				default:
					continue
				}
			}

			m.campsMut.Lock()
			if r, ok := m.campRates[msg.Campaign.ID]; ok {
				r.Incr(1)
			}
			m.campsMut.Unlock()

		// Arbitrary message.
		case msg, ok := <-m.msgQueue:
			if !ok {
				return
			}

			err := m.messengers[msg.Messenger].Push(messenger.Message{
				From:        msg.From,
				To:          msg.To,
				Subject:     msg.Subject,
				ContentType: msg.ContentType,
				Body:        msg.Body,
				AltBody:     msg.AltBody,
				Subscriber:  msg.Subscriber,
				Campaign:    msg.Campaign,
			})
			if err != nil {
				m.logger.Printf("error sending message '%s': %v", msg.Subject, err)
			}
		}
	}
}

// TemplateFuncs returns the template functions to be applied into
// compiled campaign templates.
func (m *Manager) TemplateFuncs(c *models.Campaign) template.FuncMap {
	f := template.FuncMap{
		"TrackLink": func(url string, msg *CampaignMessage) string {
			subUUID := msg.Subscriber.UUID
			if !m.cfg.IndividualTracking {
				subUUID = dummyUUID
			}

			return m.trackLink(url, msg.Campaign.UUID, subUUID)
		},
		"TrackView": func(msg *CampaignMessage) template.HTML {
			subUUID := msg.Subscriber.UUID
			if !m.cfg.IndividualTracking {
				subUUID = dummyUUID
			}

			return template.HTML(fmt.Sprintf(`<img src="%s" alt="" />`,
				fmt.Sprintf(m.cfg.ViewTrackURL, msg.Campaign.UUID, subUUID)))
		},
		"UnsubscribeURL": func(msg *CampaignMessage) string {
			return msg.unsubURL
		},
		"ManageURL": func(msg *CampaignMessage) string {
			return msg.unsubURL + "?manage=true"
		},
		"OptinURL": func(msg *CampaignMessage) string {
			// Add list IDs.
			// TODO: Show private lists list on optin e-mail
			return fmt.Sprintf(m.cfg.OptinURL, msg.Subscriber.UUID, "")
		},
		"MessageURL": func(msg *CampaignMessage) string {
			return fmt.Sprintf(m.cfg.MessageURL, c.UUID, msg.Subscriber.UUID)
		},
		"ArchiveURL": func() string {
			return m.cfg.ArchiveURL
		},
	}

	for k, v := range m.tplFuncs {
		f[k] = v
	}

	return f
}

func (m *Manager) GenericTemplateFuncs() template.FuncMap {
	return m.tplFuncs
}

// Close closes and exits the campaign manager.
func (m *Manager) Close() {
	close(m.subFetchQueue)
	close(m.campMsgErrorQueue)
	close(m.msgQueue)
}

// scanCampaigns is a blocking function that periodically scans the data source
// for campaigns to process and dispatches them to the manager.
func (m *Manager) scanCampaigns(tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()

	for {
		select {
		// Periodically scan the data source for campaigns to process.
		case <-t.C:
			campaigns, err := m.store.NextCampaigns(m.getPendingCampaignIDs())
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
		case e, ok := <-m.campMsgErrorQueue:
			if !ok {
				return
			}
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
	if _, ok := m.messengers[c.Messenger]; !ok {
		m.store.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return fmt.Errorf("unknown messenger %s on campaign %s", c.Messenger, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		return err
	}

	// Add the campaign to the active map.
	m.campsMut.Lock()
	m.camps[c.ID] = c
	m.campRates[c.ID] = ratecounter.NewRateCounter(time.Minute)
	m.campsMut.Unlock()
	return nil
}

// getPendingCampaignIDs returns the IDs of campaigns currently being processed.
func (m *Manager) getPendingCampaignIDs() []int64 {
	// Needs to return an empty slice in case there are no campaigns.
	m.campsMut.RLock()
	ids := make([]int64, 0, len(m.camps))
	for _, c := range m.camps {
		ids = append(ids, int64(c.ID))
	}
	m.campsMut.RUnlock()
	return ids
}

// nextSubscribers processes the next batch of subscribers in a given campaign.
// It returns a bool indicating whether any subscribers were processed
// in the current batch or not. A false indicates that all subscribers
// have been processed, or that a campaign has been paused or cancelled.
func (m *Manager) nextSubscribers(c *models.Campaign, batchSize int) (bool, error) {
	// Fetch a batch of subscribers.
	subs, err := m.store.NextSubscribers(c.ID, batchSize)
	if err != nil {
		return false, fmt.Errorf("error fetching campaign subscribers (%s): %v", c.Name, err)
	}

	// There are no subscribers.
	if len(subs) == 0 {
		return false, nil
	}

	// Is there a sliding window limit configured?
	hasSliding := m.cfg.SlidingWindow &&
		m.cfg.SlidingWindowRate > 0 &&
		m.cfg.SlidingWindowDuration.Seconds() > 1

	// Push messages.
	for _, s := range subs {
		// Send the message.
		msg, err := m.NewCampaignMessage(c, s)
		if err != nil {
			m.logger.Printf("error rendering message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		m.campMsgQueue <- msg

		// Check if the sliding window is active.
		if hasSliding {
			diff := time.Now().Sub(m.slidingWindowStart)

			// Window has expired. Reset the clock.
			if diff >= m.cfg.SlidingWindowDuration {
				m.slidingWindowStart = time.Now()
				m.slidingWindowNumMsg = 0
				continue
			}

			// Have the messages exceeded the limit?
			m.slidingWindowNumMsg++
			if m.slidingWindowNumMsg >= m.cfg.SlidingWindowRate {
				wait := m.cfg.SlidingWindowDuration - diff

				m.logger.Printf("messages exceeded (%d) for the window (%v since %s). Sleeping for %s.",
					m.slidingWindowNumMsg,
					m.cfg.SlidingWindowDuration,
					m.slidingWindowStart.Format(time.RFC822Z),
					wait.Round(time.Second)*1)

				m.slidingWindowNumMsg = 0
				time.Sleep(wait)
			}
		}
	}

	return true, nil
}

// isCampaignProcessing checks if the campaign is being processed.
func (m *Manager) isCampaignProcessing(id int) bool {
	m.campsMut.RLock()
	_, ok := m.camps[id]
	m.campsMut.RUnlock()
	return ok
}

func (m *Manager) exhaustCampaign(c *models.Campaign, status string) (*models.Campaign, error) {
	m.campsMut.Lock()
	delete(m.camps, c.ID)
	delete(m.campRates, c.ID)
	m.campsMut.Unlock()

	// A status has been passed. Change the campaign's status
	// without further checks.
	if status != "" {
		if err := m.store.UpdateCampaignStatus(c.ID, status); err != nil {
			m.logger.Printf("error updating campaign (%s) status to %s: %v", c.Name, status, err)
		} else {
			m.logger.Printf("set campaign (%s) to %s", c.Name, status)
		}
		return c, nil
	}

	// Fetch the up-to-date campaign status from the source.
	cm, err := m.store.GetCampaign(c.ID)
	if err != nil {
		return nil, err
	}

	// If a running campaign has exhausted subscribers, it's finished.
	if cm.Status == models.CampaignStatusRunning {
		cm.Status = models.CampaignStatusFinished
		if err := m.store.UpdateCampaignStatus(c.ID, models.CampaignStatusFinished); err != nil {
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
	url = strings.ReplaceAll(url, "&amp;", "&")

	m.linksMut.RLock()
	if uu, ok := m.links[url]; ok {
		m.linksMut.RUnlock()
		return fmt.Sprintf(m.cfg.LinkTrackURL, uu, campUUID, subUUID)
	}
	m.linksMut.RUnlock()

	// Register link.
	uu, err := m.store.CreateLink(url)
	if err != nil {
		m.logger.Printf("error registering tracking for link '%s': %v", url, err)

		// If the registration fails, fail over to the original URL.
		return url
	}

	m.linksMut.Lock()
	m.links[url] = uu
	m.linksMut.Unlock()

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

// render takes a Message, executes its pre-compiled Campaign.Tpl
// and applies the resultant bytes to Message.body to be used in messages.
func (m *CampaignMessage) render() error {
	out := bytes.Buffer{}

	// Render the subject if it's a template.
	if m.Campaign.SubjectTpl != nil {
		if err := m.Campaign.SubjectTpl.ExecuteTemplate(&out, models.ContentTpl, m); err != nil {
			return err
		}
		m.subject = out.String()
		out.Reset()
	}

	// Compile the main template.
	if err := m.Campaign.Tpl.ExecuteTemplate(&out, models.BaseTpl, m); err != nil {
		return err
	}
	m.body = out.Bytes()

	// Is there an alt body?
	if m.Campaign.ContentType != models.CampaignContentTypePlain && m.Campaign.AltBody.Valid {
		if m.Campaign.AltBodyTpl != nil {
			b := bytes.Buffer{}
			if err := m.Campaign.AltBodyTpl.ExecuteTemplate(&b, models.ContentTpl, m); err != nil {
				return err
			}
			m.altBody = b.Bytes()
		} else {
			m.altBody = []byte(m.Campaign.AltBody.String)
		}
	}

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

// AltBody returns a copy of the message's alt body.
func (m *CampaignMessage) AltBody() []byte {
	out := make([]byte, len(m.altBody))
	copy(out, m.altBody)
	return out
}

func (m *Manager) makeGnericFuncMap() template.FuncMap {
	f := template.FuncMap{
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
		"L": func() *i18n.I18n {
			return m.i18n
		},
		"Safe": func(safeHTML string) template.HTML {
			return template.HTML(safeHTML)
		},
	}

	for k, v := range sprig.GenericFuncMap() {
		f[k] = v
	}

	return f
}
