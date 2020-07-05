package media

import (
	"io"

	"gopkg.in/volatiletech/null.v6"
)

// Media represents an uploaded object.
type Media struct {
	ID        int       `db:"id" json:"id"`
	UUID      string    `db:"uuid" json:"uuid"`
	Filename  string    `db:"filename" json:"filename"`
	Thumb     string    `db:"thumb" json:"thumb"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`
	ThumbURL  string    `json:"thumb_url"`
	Provider  string    `json:"provider"`
	URL       string    `json:"url"`
}

// Store represents functions to store and retrieve media (files).
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	Get(string) string
}
