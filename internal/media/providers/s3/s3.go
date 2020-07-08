package s3

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/media"
	"github.com/rhnvrm/simples3"
)

const amznS3PublicURL = "https://%s.s3.%s.amazonaws.com%s"

// Opts represents AWS S3 specific params
type Opts struct {
	AccessKey  string        `koanf:"aws_access_key_id"`
	SecretKey  string        `koanf:"aws_secret_access_key"`
	Region     string        `koanf:"aws_default_region"`
	Bucket     string        `koanf:"bucket"`
	BucketPath string        `koanf:"bucket_path"`
	BucketURL  string        `koanf:"bucket_url"`
	BucketType string        `koanf:"bucket_type"`
	Expiry     time.Duration `koanf:"expiry"`
}

// Client implements `media.Store` for S3 provider
type Client struct {
	s3   *simples3.S3
	opts Opts
}

// NewS3Store initialises store for S3 provider. It takes in the AWS configuration
// and sets up the `simples3` client to interact with AWS APIs for all bucket operations.
func NewS3Store(opts Opts) (media.Store, error) {
	var s3svc *simples3.S3
	var err error
	if opts.Region == "" {
		return nil, errors.New("Invalid AWS Region specified. Please check `upload.s3` config")
	}
	// Use Access Key/Secret Key if specified in config.
	if opts.AccessKey != "" && opts.SecretKey != "" {
		s3svc = simples3.New(opts.Region, opts.AccessKey, opts.SecretKey)
	} else {
		// fallback to IAM role if no access key/secret key is provided.
		s3svc, err = simples3.NewUsingIAM(opts.Region)
		if err != nil {
			return nil, err
		}
	}
	return &Client{
		s3:   s3svc,
		opts: opts,
	}, nil
}

// Put takes in the filename, the content type and file object itself and uploads to S3.
func (c *Client) Put(name string, cType string, file io.ReadSeeker) (string, error) {
	// Upload input parameters
	upParams := simples3.UploadInput{
		Bucket:      c.opts.Bucket,
		ContentType: cType,
		FileName:    name,
		Body:        file,

		// Paths inside the bucket should not start with /.
		ObjectKey: strings.TrimPrefix(makeBucketPath(c.opts.BucketPath, name), "/"),
	}
	// Perform an upload.
	if _, err := c.s3.FileUpload(upParams); err != nil {
		return "", err
	}
	return name, nil
}

// Get accepts the filename of the object stored and retrieves from S3.
func (c *Client) Get(name string) string {
	// Generate a private S3 pre-signed URL if it's a private bucket.
	if c.opts.BucketType == "private" {
		url := c.s3.GeneratePresignedURL(simples3.PresignedInput{
			Bucket:        c.opts.Bucket,
			ObjectKey:     makeBucketPath(c.opts.BucketPath, name),
			Method:        "GET",
			Timestamp:     time.Now(),
			ExpirySeconds: int(c.opts.Expiry.Seconds()),
		})
		return url
	}

	// Generate a public S3 URL if it's a public bucket.
	url := ""
	if c.opts.BucketURL != "" {
		url = c.opts.BucketURL + makeBucketPath(c.opts.BucketPath, name)
	} else {
		url = fmt.Sprintf(amznS3PublicURL, c.opts.Bucket, c.opts.Region,
			makeBucketPath(c.opts.BucketPath, name))
	}
	return url
}

// Delete accepts the filename of the object and deletes from S3.
func (c *Client) Delete(name string) error {
	err := c.s3.FileDelete(simples3.DeleteInput{
		Bucket:    c.opts.Bucket,
		ObjectKey: strings.TrimPrefix(makeBucketPath(c.opts.BucketPath, name), "/"),
	})
	return err
}

func makeBucketPath(bucketPath string, name string) string {
	if bucketPath == "/" {
		return "/" + name
	}
	return fmt.Sprintf("%s/%s", bucketPath, name)
}
