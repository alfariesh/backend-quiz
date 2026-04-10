package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

var _ domain.ObjectStore = (*R2Client)(nil)

type R2Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Client(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) *R2Client {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		BaseEndpoint: &endpoint,
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	})

	return &R2Client{
		client:    client,
		bucket:    bucket,
		publicURL: publicURL,
	}
}

func (r *R2Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("uploading to R2: %w", err)
	}

	url := fmt.Sprintf("%s/%s", r.publicURL, key)
	return url, nil
}

func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from R2: %w", err)
	}
	return nil
}
