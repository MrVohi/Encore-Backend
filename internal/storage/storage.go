package storage

import (
	"context"
	"io"
)

// Driver defines a minimal object storage interface.
type Driver interface {
	Name() string
	Put(ctx context.Context, objectKey string, body io.Reader, contentType string) error
	Delete(ctx context.Context, objectKey string) error
}
