package email

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/smtppool/v2"
)

const (
	MessengerName = "email"

	hdrReturnPath = "Return-Path"
	hdrBcc        = "Bcc"
	hdrCc         = "Cc"
	hdrMessageID  = "Message-Id"
)

// Server represents an SMTP server's credentials.
type Server struct {
	// Name is a unique identifier for the server.
	Name          string            `json:"name"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`
	FromAddresses []string          `json:"from_addresses"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	//lint:ignore SA5008 ,squash is needed by koanf/mapstructure config unmarshal.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	name string

	// pools holds groups of SMTP servers indexed by a key ('from'-address
	// or a domain set per SMTPs server). An empty key holds all servers
	// and is the fallback round-robin when there's no match (old behaviour).
	pools map[string][]*Server
}

// NormalizeAddr normalizes an e-mail address (strip spaces, lowercase).
func NormalizeAddr(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// New returns an SMTP e-mail Messenger backend with the given SMTP servers.
// Group indicates whether the messenger represents a group of SMTP servers (1 or more)
// that are used as a round-robin pool, or a single server.
func New(name string, servers ...Server) (*Emailer, error) {
	e := &Emailer{
		name:  name,
		pools: make(map[string][]*Server),
	}

	for _, srv := range servers {
		s := srv

		var auth smtp.Auth
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
		case "", "none":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}
		s.Opt.Auth = auth

		// TLS config.
		s.Opt.SSL = smtppool.SSLNone
		if s.TLSType != "none" {
			s.TLSConfig = &tls.Config{}
			if s.TLSSkipVerify {
				s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
			} else {
				s.TLSConfig.ServerName = s.Host
			}

			// SSL/TLS, not STARTTLS.
			switch s.TLSType {
			case "TLS":
				s.Opt.SSL = smtppool.SSLTLS
			case "STARTTLS":
				s.Opt.SSL = smtppool.SSLSTARTTLS
			}
		}

		pool, err := smtppool.New(s.Opt)
		if err != nil {
			return nil, err
		}

		s.pool = pool

		// Add to the global list (empty key) and to each from-address
		// bucket. Duplicate keys across servers are fine and get round-robin'd.
		e.pools[""] = append(e.pools[""], &s)
		for _, addr := range s.FromAddresses {
			if key := NormalizeAddr(addr); key != "" {
				e.pools[key] = append(e.pools[key], &s)
			}
		}
	}

	return e, nil
}

// Name returns the messenger's name.
func (e *Emailer) Name() string {
	return e.name
}

// Push pushes a message to the server.
func (e *Emailer) Push(m models.Message) error {
	// Pick the from-address-routed pool if there is one, else default
	// to the full pool (empty key) for roundrobin.
	pool := e.pools[""]
	if len(e.pools) > 1 {
		if srvs := e.getPool(m.From); srvs != nil {
			pool = srvs
		}
	}
	srv := pool[rand.Intn(len(pool))]

	// Are there attachments?
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	// Create the email.
	em := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Subject:     m.Subject,
		Attachments: files,
	}

	em.Headers = textproto.MIMEHeader{}

	// Attach SMTP level headers.
	for k, v := range srv.EmailHeaders {
		em.Headers.Set(k, v)
	}

	// Attach e-mail level headers.
	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	// Generate Message-Id based on the From address.
	if em.Headers.Get(hdrMessageID) == "" {
		d := "localhost"
		if a, err := mail.ParseAddress(m.From); err == nil {
			d = a.Address[strings.LastIndex(a.Address, "@")+1:]
		}
		if r, err := utils.GenerateRandomString(24); err == nil {
			em.Headers.Set(hdrMessageID, fmt.Sprintf("<%s@%s>", r, d))
		}
	}

	// If the `Return-Path` header is set, it should be set as the
	// the SMTP envelope sender (via the Sender field of the email struct).
	if sender := em.Headers.Get(hdrReturnPath); sender != "" {
		em.Sender = sender
		em.Headers.Del(hdrReturnPath)
	}

	// If the `Bcc` header is set, it should be set on the Envelope
	if bcc := em.Headers.Get(hdrBcc); bcc != "" {
		for _, part := range strings.Split(bcc, ",") {
			em.Bcc = append(em.Bcc, strings.TrimSpace(part))
		}
		em.Headers.Del(hdrBcc)
	}

	// If the `Cc` header is set, it should be set on the Envelope
	if cc := em.Headers.Get(hdrCc); cc != "" {
		for _, part := range strings.Split(cc, ",") {
			em.Cc = append(em.Cc, strings.TrimSpace(part))
		}
		em.Headers.Del(hdrCc)
	}

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Body)
	default:
		em.HTML = m.Body
		if len(m.AltBody) > 0 {
			em.Text = m.AltBody
		}
	}

	return srv.pool.Send(em)
}

// Flush flushes the message queue to the server.
func (e *Emailer) Flush() error {
	return nil
}

// Close closes the SMTP pools.
func (e *Emailer) Close() error {
	for _, s := range e.pools[""] {
		s.pool.Close()
	}
	return nil
}

// getPool returns the pool of servers configured to handle the given From
// header, matched by full e-mail and then by domain.
// Returns nil if no mapping matches.
func (e *Emailer) getPool(from string) []*Server {
	addr := utils.ParseEmailAddress(from)
	if addr == "" {
		return nil
	}

	if srvs, ok := e.pools[addr]; ok {
		return srvs
	}

	if _, after, ok := strings.Cut(addr, "@"); ok {
		return e.pools[after]
	}

	return nil
}
