package s3store

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	URL    string
	Region string
	ID     string
	Secret string
	Bucket string
}

type Store struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func New(c Config) (*Store, error) {
	client := s3.NewFromConfig(aws.Config{Region: c.Region}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.URL)
		o.Credentials = credentials.NewStaticCredentialsProvider(c.ID, c.Secret, "")
		o.UsePathStyle = true
	})

	presignClient := s3.NewPresignClient(client)

	return &Store{
		client:        client,
		presignClient: presignClient,
		bucket:        c.Bucket,
	}, nil
}

func (s *Store) NewUploadURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	}
	presigned, err := s.presignClient.PresignPutObject(ctx, params, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return presigned.URL, nil
}
