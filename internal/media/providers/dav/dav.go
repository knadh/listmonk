package dav

import (
	"encoding/base64"
	"io"
	"path/filepath"
	"strings"

	"github.com/knadh/listmonk/internal/media"
	"github.com/studio-b12/gowebdav"
)

// Opts represents WebDAV specific params
type Opts struct {
	Endpoint string            `koanf:"endpoint"`
	Username string            `koanf:"username"`
	Password string            `koanf:"password"`
	RootPath string            `koanf:"root_path"`
	Headers  map[string]string `koanf:"headers"`
}

// Client implimenting media.Store interface
type Client struct {
	root   string
	client *gowebdav.Client
}

// Connect tests connection request.
func Connect(o Opts) bool {
	c := gowebdav.NewClient(o.Endpoint, o.Username, o.Password)

	for k, v := range o.Headers {
		c.SetHeader(k, v)
	}

	if err := c.Connect(); err != nil {
		return false
	}

	return true
}

// New will create new WebDAV Storage.
func New(o Opts) (media.Store, error) {
	if o.RootPath == "" {
		o.RootPath = "/"
	}

	c := &Client{
		client: gowebdav.NewClient(o.Endpoint, o.Username, o.Password),
		root:   o.RootPath,
	}

	for k, v := range o.Headers {
		c.client.SetHeader(k, v)
	}

	if err := c.client.Connect(); err != nil {
		return nil, err
	}

	return c, nil
}

// Put takes in the filename, the content type and file object itself and uploads WebDav.
func (c *Client) Put(path, ct string, r io.ReadSeeker) (string, error) {
	fullPath := c.pathTo(path)

	// Create directories for the requested path.
	if err := c.mkDirAll(fullPath); err != nil {
		return "", err
	}

	// Write file.
	if err := c.client.WriteStream(c.pathTo(path), r, 0644); err != nil {
		return "", err
	}

	// Return full path file has uploaded.
	return fullPath, nil
}

// Delete file from WebDAV Storage.
func (c *Client) Delete(path string) error {
	return c.client.Remove(c.pathTo(path))
}

// Get file from WebDAV Storage.
func (c *Client) Get(path string) string {
	body, err := c.client.Read(path)
	if err != nil {
		return ""
	}

	return base64.RawStdEncoding.EncodeToString(body)
}

func (c *Client) pathTo(path string) string {
	if strings.HasPrefix(path, c.root) {
		return path
	}

	return filepath.Join(c.root, path)
}

func (c *Client) mkDirAll(path string) error {
	if filepath.Dir(path) == "/" {
		return nil
	}

	return c.client.MkdirAll(filepath.Dir(path), 0644)
}
