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
	servers        []*Server
	name           string
	fromAddrToSrv  map[string][]*Server
}

// New returns an SMTP e-mail Messenger backend with the given SMTP servers.
// Group indicates whether the messenger represents a group of SMTP servers (1 or more)
// that are used as a round-robin pool, or a single server.
func New(name string, servers ...Server) (*Emailer, error) {
	e := &Emailer{
		servers:       make([]*Server, 0, len(servers)),
		name:          name,
		fromAddrToSrv: make(map[string][]*Server),
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
		e.servers = append(e.servers, &s)

		// Map from addresses to this server.
		// Track which keys are already mapped to this server to avoid duplicates.
		seenForServer := make(map[string]bool)
		for _, addr := range s.FromAddresses {
			key := strings.ToLower(strings.TrimSpace(addr))
			// Skip empty keys and duplicates for this server.
			if key == "" {
				continue
			}
			if seenForServer[key] {
				continue
			}
			seenForServer[key] = true
			e.fromAddrToSrv[key] = append(e.fromAddrToSrv[key], &s)
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
	// Check if there's a specific SMTP server mapped to the from address.
	var (
		ln  = len(e.servers)
		srv *Server
	)

	// Extract the email address from the From field using RFC 5322 parsing.
	fromAddr := m.From
	if addr, err := mail.ParseAddress(m.From); err == nil {
		fromAddr = addr.Address
	}
	// If parsing fails, fall back to the raw From value (for backward compatibility).
	fromAddr = strings.ToLower(strings.TrimSpace(fromAddr))

	// Check if there's a server mapped to this from address.
	// First try exact match, then try domain match.
	var matchedServers []*Server
	if servers, ok := e.fromAddrToSrv[fromAddr]; ok {
		matchedServers = servers
	} else {
		// Extract domain from email address for domain matching.
		if atIdx := strings.Index(fromAddr, "@"); atIdx >= 0 {
			domain := fromAddr[atIdx:] // includes @ symbol
			if servers, ok := e.fromAddrToSrv[domain]; ok {
				matchedServers = servers
			} else {
				// Try without @ symbol.
				domainOnly := fromAddr[atIdx+1:]
				if servers, ok := e.fromAddrToSrv[domainOnly]; ok {
					matchedServers = servers
				}
			}
		}
	}

	// If we found matching servers, pick one randomly (round-robin).
	if len(matchedServers) > 0 {
		if len(matchedServers) > 1 {
			srv = matchedServers[rand.Intn(len(matchedServers))]
		} else {
			srv = matchedServers[0]
		}
	}

	// If no match found, fall back to load balancing.
	if srv == nil {
		if ln > 1 {
			// If there are more than one SMTP servers, send to a random one from the list.
			srv = e.servers[rand.Intn(ln)]
		} else {
			srv = e.servers[0]
		}
	}

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
	for _, s := range e.servers {
		s.pool.Close()
	}
	return nil
}
