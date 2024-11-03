package storage

import (
	"bytes"
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type Service interface {
	GetClient() *s3.Client
	GetUploader() *manager.Uploader
	UploadFile(fileData []byte, fileId uuid.UUID, fileType string) (*manager.UploadOutput, error)
}

type service struct {
	client   *s3.Client
	uploader *manager.Uploader
}

var (
	bucket          = os.Getenv("S3_BUCKET")
	region          = os.Getenv("S3_REGION")
	accessKeyId     = os.Getenv("S3_ACCESS_KEY_ID")
	accessKey       = os.Getenv("S3_SECRET_ACCESS_KEY")
	session         = os.Getenv("S3_SESSION")
	storageInstance *service
)

func New() Service {
	// Reuse Connection
	if storageInstance != nil {
		return storageInstance
	}

	staticProvider := credentials.NewStaticCredentialsProvider(
		accessKeyId,
		accessKey,
		session,
	)
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(staticProvider),
	)
	if err != nil {
		log.Fatalf("load s3 config error %v", err)
	}

	client := s3.NewFromConfig(cfg)
	storageInstance = &service{
		client:   client,
		uploader: manager.NewUploader(client),
	}

	return storageInstance
}

func (s *service) GetClient() *s3.Client {
	return s.client
}

func (s *service) GetUploader() *manager.Uploader {
	return s.uploader
}

func (s *service) UploadFile(fileData []byte, fileId uuid.UUID, fileType string) (*manager.UploadOutput, error) {
	return s.uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fileId.String()),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(fileType),
	})
}
