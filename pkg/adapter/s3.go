package adapter

import (
	"bytes"
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type (
	S3SDK struct {
		Client *s3.Client
	}

	S3Adapter interface {
		UploadFile(ctx context.Context, localfileName string, bucketName string, bucketKey string) error
	}
)

// UploadFile implements S3Adapter.
func (s *S3SDK) UploadFile(ctx context.Context, localfileName string, bucketName string, bucketKey string) error {
	data, err := os.ReadFile(localfileName)
	if err != nil {
		return err
	}
	_, err = s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &bucketKey,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return err
	}
	return nil
}

func NewS3Adapter() (S3Adapter, error) {
	client, err := createS3Client(context.Background())
	if err != nil {
		return nil, err
	}
	return &S3SDK{
		Client: client,
	}, nil
}

func createS3Client(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return client, nil
}
