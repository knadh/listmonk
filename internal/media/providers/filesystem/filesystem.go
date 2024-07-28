package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

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
	var out *os.File

	// Get the directory path
	dir := getDir(c.opts.UploadPath)
	filename = assertUniqueFilename(dir, filename)
	o, err := os.OpenFile(filepath.Join(dir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return "", err
	}
	out = o
	defer out.Close()

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
	if err != nil {
		return err
	}
	return nil
}

// assertUniqueFilename takes a file path and checks if it exists on the disk.
// If it doesn't, it returns the same name. If it does, it adds a numeric suffix and returns the new name.
//
// Example:
//
//	If a file `uploads/my-image_1.jpg` already exists on the disk,
//	the function would return `uploads/my-image_2.jpg` for a new file with the same name.
func assertUniqueFilename(dir, fileName string) string {
	var (
		ext  = filepath.Ext(fileName)
		base = fileName[0 : len(fileName)-len(ext)]
		num  = 0
	)

	for {
		// There's no name conflict.
		if _, err := os.Stat(filepath.Join(dir, fileName)); os.IsNotExist(err) {
			return fileName
		}

		// Does the name match the _(num) syntax?
		r := media.FnameRegexp.FindAllStringSubmatch(fileName, -1)
		if len(r) == 1 && len(r[0]) == 3 {
			num, _ = strconv.Atoi(r[0][2])
		}
		num++

		fileName = fmt.Sprintf("%s_%d%s", base, num, ext)
	}
}

// getDir returns the current working directory path if no directory is specified,
// else returns the directory path specified itself.
func getDir(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return dir
}
