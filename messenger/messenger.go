package messenger

// Messenger is an interface for a generic messaging backend,
// for instance, e-mail, SMS etc.
type Messenger interface {
	Name() string

	Push(fromAddr, toAddr, subject string, message []byte) error
	Flush() error
}
