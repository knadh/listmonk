package mailbox

import "time"

// Maximum message size to download
var maxMessageSize = 1024 * 1024 // 1mb

// Opt represents an e-mail POP/IMAP mailbox configuration.
type Opt struct {
	// Host is the server's hostname.
	Host string `json:"host"`

	// Port is the server port.
	Port int `json:"port"`

	AuthProtocol string `json:"auth_protocol"`

	// Username is the mail server login username.
	Username string `json:"username"`

	// Password is the mail server login password.
	Password string `json:"password"`

	// Folder is the name of the IMAP folder to scan for e-mails.
	Folder string `json:"folder"`

	// TrashFolder is the name of the IMAP folder to more processed e-mails to
	// instead of deleting.
	TrashFolder string `json:"trash_folder"`

	// Optional TLS settings.
	TLSEnabled    bool `json:"tls_enabled"`
	TLSSkipVerify bool `json:"tls_skip_verify"`

	ScanInterval time.Duration `json:"scan_interval"`
}
