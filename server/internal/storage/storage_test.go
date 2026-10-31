//go:build unit

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("disk backend returns a disk store", func(t *testing.T) {
		s, err := New(context.Background(), Config{Backend: "disk", Dir: t.TempDir()})

		require.NoError(t, err)
		_, ok := s.(*Disk)
		require.True(t, ok)
	})

	t.Run("unknown backend defaults to disk", func(t *testing.T) {
		s, err := New(context.Background(), Config{Dir: t.TempDir()})

		require.NoError(t, err)
		_, ok := s.(*Disk)
		require.True(t, ok)
	})

	t.Run("s3 backend returns an s3 store", func(t *testing.T) {
		s, err := New(context.Background(), Config{
			Backend:   "s3",
			Bucket:    "b",
			Region:    "us-east-1",
			AccessKey: "ak",
			SecretKey: "sk",
		})

		require.NoError(t, err)
		_, ok := s.(*S3)
		require.True(t, ok)
	})
}
