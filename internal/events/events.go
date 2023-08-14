// Package events implements a simple event broadcasting mechanism
// for usage in broadcasting error messages, postbacks etc. various
// channels.
package events

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

const (
	TypeError = "error"
)

// Event represents a single event in the system.
type Event struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Channels []string    `json:"-"`
}

type Events struct {
	subs map[string]chan Event
	sync.RWMutex
}

// New returns a new instance of Events.
func New() *Events {
	return &Events{
		subs: make(map[string]chan Event),
	}
}

// Subscribe returns a channel to which the given event `types` are streamed.
// id is the unique identifier for the caller. A caller can only register
// for subscription once.
func (ev *Events) Subscribe(id string) (chan Event, error) {
	ev.Lock()
	defer ev.Unlock()

	if ch, ok := ev.subs[id]; ok {
		return ch, nil
	}

	ch := make(chan Event, 100)
	ev.subs[id] = ch

	return ch, nil
}

// Unsubscribe unsubscribes a subscriber (obviously).
func (ev *Events) Unsubscribe(id string) {
	ev.Lock()
	defer ev.Unlock()
	delete(ev.subs, id)
}

// Publish publishes an event to all subscribers.
func (ev *Events) Publish(e Event) error {
	ev.Lock()
	defer ev.Unlock()

	for _, ch := range ev.subs {
		select {
		case ch <- e:
		default:
			return fmt.Errorf("event queue full for type: %s", e.Type)
		}
	}

	return nil
}

// This implements an io.Writer specifically for receiving error messages
// mirrored (io.MultiWriter) from error log writing.
type wri struct {
	ev *Events
}

func (w *wri) Write(b []byte) (n int, err error) {
	// Only broadcast error messages.
	if !bytes.Contains(b, []byte("error")) {
		return 0, nil
	}

	w.ev.Publish(Event{
		Type:    TypeError,
		Message: string(b),
	})

	return len(b), nil
}

func (ev *Events) ErrWriter() io.Writer {
	return &wri{ev: ev}
}
