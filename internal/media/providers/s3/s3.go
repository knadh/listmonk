package s3

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/media"
	"github.com/rhnvrm/simples3"
)

// Opt represents AWS S3 specific params
type Opt struct {
	URL        string        `koanf:"url"`
	PublicURL  string        `koanf:"public_url"`
	AccessKey  string        `koanf:"aws_access_key_id"`
	SecretKey  string        `koanf:"aws_secret_access_key"`
	Region     string        `koanf:"aws_default_region"`
	Bucket     string        `koanf:"bucket"`
	BucketPath string        `koanf:"bucket_path"`
	BucketType string        `koanf:"bucket_type"`
	Expiry     time.Duration `koanf:"expiry"`
}

// Client implements `media.Store` for S3 provider
type Client struct {
	s3   *simples3.S3
	opts Opt
}

// NewS3Store initialises store for S3 provider. It takes in the AWS configuration
// and sets up the `simples3` client to interact with AWS APIs for all bucket operations.
func NewS3Store(opt Opt) (media.Store, error) {
	var cl *simples3.S3
	if opt.URL == "" {
		opt.URL = fmt.Sprintf("https://s3.%s.amazonaws.com", opt.Region)
	}
	opt.URL = strings.TrimRight(opt.URL, "/")

	if opt.AccessKey == "" && opt.SecretKey == "" {
		// fallback to IAM role if no access key/secret key is provided.
		cl, _ = simples3.NewUsingIAM(opt.Region)
	}

	if cl == nil {
		cl = simples3.New(opt.Region, opt.AccessKey, opt.SecretKey)
	}

	cl.SetEndpoint(opt.URL)

	return &Client{
		s3:   cl,
		opts: opt,
	}, nil
}

// Put takes in the filename, the content type and file object itself and uploads to S3.
func (c *Client) Put(name string, cType string, file io.ReadSeeker) (string, error) {
	// Upload input parameters
	p := simples3.UploadInput{
		Bucket:      c.opts.Bucket,
		ContentType: cType,
		FileName:    name,
		Body:        file,

		// Paths inside the bucket should not start with /.
		ObjectKey: c.makeBucketPath(name),
	}

	if c.opts.BucketType == "public" {
		p.ACL = "public-read"
	}

	// Upload.
	if _, err := c.s3.FilePut(p); err != nil {
		return "", err
	}
	return name, nil
}

// Get accepts the filename of the object stored and retrieves from S3.
func (c *Client) Get(name string) string {
	// Generate a private S3 pre-signed URL if it's a private bucket, and there
	// is no public URL provided.
	if c.opts.BucketType == "private" && c.opts.PublicURL == "" {
		u := c.s3.GeneratePresignedURL(simples3.PresignedInput{
			Bucket:        c.opts.Bucket,
			ObjectKey:     c.makeBucketPath(name),
			Method:        "GET",
			Timestamp:     time.Now(),
			ExpirySeconds: int(c.opts.Expiry.Seconds()),
		})
		return u
	}

	// Generate a public S3 URL if it's a public bucket or a public URL is
	// provided.
	return c.makeFileURL(name)
}

// Delete accepts the filename of the object and deletes from S3.
func (c *Client) Delete(name string) error {
	err := c.s3.FileDelete(simples3.DeleteInput{
		Bucket:    c.opts.Bucket,
		ObjectKey: c.makeBucketPath(name),
	})
	return err
}

// makeBucketPath returns the file path inside the bucket. The path should not
// start with a /.
func (c *Client) makeBucketPath(name string) string {
	// If the path is root (/), return the filename without the preceding slash.
	p := strings.TrimPrefix(strings.TrimSuffix(c.opts.BucketPath, "/"), "/")
	if p == "" {
		return name
	}

	// whatever/bucket/path/filename.jpg: No preceding slash.
	return p + "/" + name
}

func (c *Client) makeFileURL(name string) string {
	if c.opts.PublicURL != "" {
		return c.opts.PublicURL + "/" + c.makeBucketPath(name)
	}

	return c.opts.URL + "/" + c.opts.Bucket + "/" + c.makeBucketPath(name)
}
