package service

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/domain"
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"audio/mpeg": true,
	"audio/mp4":  true,
	"audio/ogg":  true,
	"audio/wav":  true,
}

type MediaService struct {
	client        *s3.Client
	bucketName    string
	publicURL     string
	maxFileSizeMB int
}

func NewMediaService(cfg config.R2Config) *MediaService {
	if cfg.AccountID == "" {
		// R2 not configured, return a no-op service
		return &MediaService{maxFileSizeMB: cfg.MaxFileSizeMB}
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		// Log error but don't crash; media upload will fail gracefully
		return &MediaService{maxFileSizeMB: cfg.MaxFileSizeMB}
	}

	client := s3.NewFromConfig(awsCfg)

	return &MediaService{
		client:        client,
		bucketName:    cfg.BucketName,
		publicURL:     strings.TrimRight(cfg.PublicURL, "/"),
		maxFileSizeMB: cfg.MaxFileSizeMB,
	}
}

func (s *MediaService) Upload(ctx context.Context, userID uuid.UUID, fileName, mimeType string, fileSize int64, body io.Reader) (*domain.Media, error) {
	if s.client == nil {
		return nil, fmt.Errorf("media storage not configured")
	}

	if !allowedMimeTypes[mimeType] {
		return nil, domain.ErrUnsupportedMedia
	}

	maxBytes := int64(s.maxFileSizeMB) * 1024 * 1024
	if fileSize > maxBytes {
		return nil, domain.ErrFileTooLarge
	}

	ext := path.Ext(fileName)
	objectKey := fmt.Sprintf("media/%s/%s/%s%s",
		userID.String(),
		time.Now().Format("2006/01/02"),
		uuid.New().String(),
		ext,
	)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        body,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("uploading to R2: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.publicURL, objectKey)

	media := &domain.Media{
		UserID:   userID,
		FileName: fileName,
		FileSize: int(fileSize),
		MimeType: mimeType,
		R2Key:    objectKey,
		URL:      publicURL,
	}

	return media, nil
}

func (s *MediaService) Delete(ctx context.Context, r2Key string) error {
	if s.client == nil {
		return fmt.Errorf("media storage not configured")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(r2Key),
	})
	return err
}
