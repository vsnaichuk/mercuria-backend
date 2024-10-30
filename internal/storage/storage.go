package storage

import (
	"os"

	"github.com/gofiber/storage/s3/v2"
)

var (
	bucket   = os.Getenv("S3_BUCKET")
	endpoint = os.Getenv("S3_ENDPOINT")
	region   = os.Getenv("S3_REGION")
	S3       *s3.Storage
)

func New() *s3.Storage {
	S3 = s3.New(s3.Config{
		Bucket:   bucket,
		Endpoint: endpoint,
		Region:   region,
		Reset:    false,
	})
	return S3
}
