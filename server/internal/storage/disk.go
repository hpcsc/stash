package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var _ Storage = (*Disk)(nil)

type Disk struct {
	dir string
}

func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage directory %s: %w", dir, err)
	}
	return &Disk{dir: dir}, nil
}

func (d *Disk) Save(_ context.Context, key string, reader io.Reader) (string, error) {
	path := filepath.Join(d.dir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("creating parent directory for %s: %w", key, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("creating file %s: %w", key, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("writing file %s: %w", key, err)
	}

	return key, nil
}

func (d *Disk) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(d.dir, key))
}