package s3store

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
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
	client          *s3.Client
	presignClient   *s3.PresignClient
	transferManager *transfermanager.Client
	bucket          string
}

func New(c Config) (*Store, error) {
	client := s3.NewFromConfig(aws.Config{Region: c.Region}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.URL)
		o.Credentials = credentials.NewStaticCredentialsProvider(c.ID, c.Secret, "")
		o.UsePathStyle = true
	})

	transferManager := transfermanager.New(client, func(o *transfermanager.Options) {
		o.PartSizeBytes = 16 * 1024 * 1024
		o.Concurrency = 4
	})

	return &Store{
		client:          client,
		presignClient:   s3.NewPresignClient(client),
		transferManager: transferManager,
		bucket:          c.Bucket,
	}, nil
}

func (s *Store) Download(ctx context.Context, id string, w io.Writer) error {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, out.Body)
	return err
}

func (s *Store) NewDownloadURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	presignedUrl, err := s.presignClient.PresignGetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(id),
		},
		s3.WithPresignExpires(ttl),
	)
	if err != nil {
		return "", err
	}
	return presignedUrl.URL, nil
}

func (s *Store) Upload(ctx context.Context, id string, r io.Reader) error {
	_, err := s.transferManager.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
		Body:   r,
	})
	return err
}

func (s *Store) NewUploadURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	presigned, err := s.presignClient.PresignPutObject(ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(id),
		},
		s3.WithPresignExpires(ttl),
	)
	if err != nil {
		return "", err
	}
	return presigned.URL, nil
}
