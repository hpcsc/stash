package storage

import (
	"context"
	"io"
)

type Storage interface {
	Save(ctx context.Context, key string, reader io.Reader) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// New builds the storage backend named by cfg.Backend: "s3" or "disk" (the default).
func New(ctx context.Context, cfg Config) (Storage, error) {
	switch cfg.Backend {
	case "s3":
		return NewS3(ctx, cfg)
	default:
		return NewDisk(cfg.Dir)
	}
}
