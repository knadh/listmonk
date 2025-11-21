package models

import (
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

const (
	SubscriberStatusEnabled     = "enabled"
	SubscriberStatusDisabled    = "disabled"
	SubscriberStatusBlockListed = "blocklisted"
)

// Subscribers represents a slice of Subscriber.
type Subscribers []Subscriber

// Subscriber represents an e-mail subscriber.
type Subscriber struct {
	Base

	UUID    string         `db:"uuid" json:"uuid"`
	Email   string         `db:"email" json:"email" form:"email"`
	Name    string         `db:"name" json:"name" form:"name"`
	Attribs JSON           `db:"attribs" json:"attribs"`
	Status  string         `db:"status" json:"status"`
	Lists   types.JSONText `db:"lists" json:"lists"`
}

// GetIDs returns the list of subscriber IDs.
func (subs Subscribers) GetIDs() []int {
	IDs := make([]int, len(subs))
	for i, c := range subs {
		IDs[i] = c.ID
	}

	return IDs
}

// LoadLists lazy loads the lists for all the subscribers
// in the Subscribers slice and attaches them to their []Lists property.
func (subs Subscribers) LoadLists(stmt *sqlx.Stmt) error {
	var sl []subLists
	err := stmt.Select(&sl, pq.Array(subs.GetIDs()))
	if err != nil {
		return err
	}

	if len(subs) != len(sl) {
		return errors.New("campaign stats count does not match")
	}

	for i, s := range sl {
		if s.SubscriberID == subs[i].ID {
			subs[i].Lists = s.Lists
		}
	}

	return nil
}

// FirstName splits the name by spaces and returns the first chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's first name.
func (s Subscriber) FirstName() string {
	for _, s := range strings.Split(s.Name, " ") {
		if len(s) > 2 {
			return s
		}
	}

	return s.Name
}

// LastName splits the name by spaces and returns the last chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's last name.
func (s Subscriber) LastName() string {
	chunks := strings.Split(s.Name, " ")
	for i := len(chunks) - 1; i >= 0; i-- {
		chunk := chunks[i]
		if len(chunk) > 2 {
			return chunk
		}
	}

	return s.Name
}
