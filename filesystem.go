package filesystem

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

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

	// Ensure sub-directories exist when filename contains folders.
	subdir := filepath.Dir(filename)
	if subdir != "." && subdir != "" {
		// Use filepath.Join so OS-specific separators are handled.
		if err := os.MkdirAll(filepath.Join(dir, subdir), 0755); err != nil {
			return "", err
		}
	}

	// Read the file contents.
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
	// Determine the relative path of the file inside the upload path.
	rel := url

	// If the value looks like a full URL that contains RootURL and UploadURI, trim them.
	// Normalize to forward slashes for URL path handling.
	rel = path.Clean(rel)
	// Remove root URL and upload URI if present.
	if c.opts.RootURL != "" && len(rel) >= len(c.opts.RootURL) && rel[:len(c.opts.RootURL)] == c.opts.RootURL {
		rel = rel[len(c.opts.RootURL):]
	}
	if c.opts.UploadURI != "" && len(rel) >= len(c.opts.UploadURI) && rel[:len(c.opts.UploadURI)] == c.opts.UploadURI {
		rel = rel[len(c.opts.UploadURI):]
	}

	rel = path.Clean(rel)
	rel = strings.TrimPrefix(rel, "/")

	// Join with the upload path and read the file.
	full := filepath.Join(getDir(c.opts.UploadPath), filepath.FromSlash(rel))
	b, err := os.ReadFile(full)
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
