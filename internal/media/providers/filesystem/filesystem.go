package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/knadh/listmonk/internal/media"
)

// Opts represents filesystem params
type Opts struct {
	UploadPath string `koanf:"upload_path"`
	UploadURI  string `koanf:"upload_uri"`
	RootURL    string `koanf:"root_url"`
}

// Client implements `media.Store`
type Client struct {
	opts Opts
}

// New initialises store for Filesystem provider.
func New(opts Opts) (media.Store, error) {
	return &Client{
		opts: opts,
	}, nil
}

// Put accepts the filename, the content type and file object itself and stores the file in disk.
func (c *Client) Put(filename string, cType string, src io.ReadSeeker) (string, error) {
	// Get the directory path
	dir := getDir(c.opts.UploadPath)

	// Read the  file contents.
	out, err := os.OpenFile(filepath.Join(dir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Copy it to the target location.
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}

	return filename, nil
}

// GetURL accepts a filename and retrieves the full path from disk.
func (c *Client) GetURL(name string) string {
	return fmt.Sprintf("%s%s/%s", c.opts.RootURL, c.opts.UploadURI, name)
}

// GetBlob accepts a URL, reads the file, and returns the blob.
func (c *Client) GetBlob(url string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(getDir(c.opts.UploadPath), filepath.Base(url)))
	return b, err
}

// Delete accepts a filename and removes it from disk.
func (c *Client) Delete(file string) error {
	dir := getDir(c.opts.UploadPath)

	err := os.Remove(filepath.Join(dir, file))
	return err
}

// getDir returns the current working directory path if no directory is specified,
// else returns the directory path specified itself.
func getDir(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}

	return dir
}
