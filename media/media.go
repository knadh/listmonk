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
	Width     int       `db:"width" json:"width"`
	Height    int       `db:"height" json:"height"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`
	ThumbURI  string    `json:"thumb_uri"`
	URI       string    `json:"uri"`
}

// Store represents set of methods to perform upload/delete operations.
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	Get(string) string
}
