package storage

import (
	"context"
	"io"
)

type Storage interface {
	Save(ctx context.Context, key string, reader io.Reader) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}