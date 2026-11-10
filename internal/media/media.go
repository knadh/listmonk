package media

import (
	"io"

	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	"gopkg.in/volatiletech/null.v6"
)

// Storage provider names.
const (
	ProviderS3         = "s3"
	ProviderFilesystem = "filesystem"
)

// Media represents an uploaded object.
type Media struct {
	ID          int            `db:"id" json:"id"`
	UUID        string         `db:"uuid" json:"uuid"`
	Filename    string         `db:"filename" json:"filename"`
	ContentType string         `db:"content_type" json:"content_type"`
	Thumb       string         `db:"thumb" json:"-"`
	CreatedAt   null.Time      `db:"created_at" json:"created_at"`
	ThumbURL    null.String    `json:"thumb_url"`
	Provider    string         `json:"provider"`
	Meta        models.JSON    `db:"meta" json:"meta"`
	URL         string         `json:"url"`
	Tags        pq.StringArray `db:"tags" json:"tags"`

	Total int `db:"total" json:"-"`
}

// Tag represents a tag and item counts using it.
type Tag struct {
	Tag   string `db:"tag" json:"tag"`
	Count int    `db:"count" json:"count"`
}

// Store represents functions to store and retrieve media (files).
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	GetURL(string) string
	GetBlob(string) ([]byte, error)
}
