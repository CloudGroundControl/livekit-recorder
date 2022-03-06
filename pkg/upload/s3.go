package upload

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Region    string
	Bucket    string
	Directory string
}

var ErrEmptyS3BucketName = errors.New("empty S3 bucket name")

type s3Uploader struct {
	bucket    string
	directory string
	service   *manager.Uploader
}

func NewS3Uploader(config S3Config) (Uploader, error) {
	// Create a TODO context
	ctx := context.TODO()

	// Load S3 config
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, err
	}

	// Create service
	service := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(service)

	return &s3Uploader{config.Bucket, config.Directory, uploader}, nil
}

func (s *s3Uploader) Upload(key string, body io.Reader) error {
	// Append directory if it's not empty
	var uploadKey = key
	if s.directory != "" {
		uploadKey = fmt.Sprintf("%s/%s", s.directory, key)
	}

	_, err := s.service.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(uploadKey),
		Body:   body,
	})
	return err
}

func (s *s3Uploader) GetDirectory() string {
	return s.directory
}
