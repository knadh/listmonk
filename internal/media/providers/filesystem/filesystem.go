package filesystem

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/knadh/listmonk/internal/media"
)

const tmpFilePrefix = "listmonk"

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

// This matches filenames, sans extensions, of the format
// filename_(number). The number is incremented in case
// new file uploads conflict with existing filenames
// on the filesystem.
var fnameRegexp = regexp.MustCompile(`(.+?)_([0-9]+)$`)

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

// assertUniqueFilename takes a file path and check if it exists on the disk. If it doesn't,
// it returns the same name and if it does, it adds a small random hash to the filename
// and returns that.
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
		r := fnameRegexp.FindAllStringSubmatch(fileName, -1)
		if len(r) == 1 && len(r[0]) == 3 {
			num, _ = strconv.Atoi(r[0][2])
		}
		num++

		fileName = fmt.Sprintf("%s_%d%s", base, num, ext)
	}
}

// generateRandomString generates a cryptographically random, alphanumeric string of length n.
func generateRandomString(n int) (string, error) {
	const dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	var bytes = make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// getDir returns the current working directory path if no directory is specified,
// else returns the directory path specified itself.
func getDir(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return dir
}
