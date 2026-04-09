package domain

import (
	"context"
	"io"
)

// ObjectStore abstracts file storage operations (R2, S3, local, etc.).
type ObjectStore interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (url string, err error)
	Delete(ctx context.Context, key string) error
}
