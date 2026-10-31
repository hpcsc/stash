//go:build unit

package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type fakeS3Client struct {
	putBucket string
	putKey    string
	putBody   []byte
	putErr    error
	getBucket string
	getKey    string
	getBody   string
	getErr    error
}

func (f *fakeS3Client) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putBucket = aws.ToString(params.Bucket)
	f.putKey = aws.ToString(params.Key)
	if params.Body != nil {
		f.putBody, _ = io.ReadAll(params.Body)
	}
	return &s3.PutObjectOutput{}, f.putErr
}

func (f *fakeS3Client) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.getBucket = aws.ToString(params.Bucket)
	f.getKey = aws.ToString(params.Key)
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(f.getBody))}, nil
}

func TestS3(t *testing.T) {
	t.Run("save uploads the object", func(t *testing.T) {
		client := &fakeS3Client{}
		s := &S3{client: client, bucket: "my-bucket"}

		key, err := s.Save(context.Background(), "images/a.png", strings.NewReader("bytes"))

		require.NoError(t, err)
		require.Equal(t, "images/a.png", key)
		require.Equal(t, "my-bucket", client.putBucket)
		require.Equal(t, "images/a.png", client.putKey)
		require.Equal(t, "bytes", string(client.putBody))
	})

	t.Run("save returns the upload error", func(t *testing.T) {
		s := &S3{client: &fakeS3Client{putErr: errors.New("boom")}, bucket: "my-bucket"}

		_, err := s.Save(context.Background(), "k", strings.NewReader("x"))

		require.Error(t, err)
	})

	t.Run("open downloads the object", func(t *testing.T) {
		client := &fakeS3Client{getBody: "bytes"}
		s := &S3{client: client, bucket: "my-bucket"}

		reader, err := s.Open(context.Background(), "images/a.png")

		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "bytes", string(body))
		require.Equal(t, "my-bucket", client.getBucket)
		require.Equal(t, "images/a.png", client.getKey)
	})

	t.Run("open returns the get error", func(t *testing.T) {
		s := &S3{client: &fakeS3Client{getErr: errors.New("missing")}, bucket: "my-bucket"}

		_, err := s.Open(context.Background(), "k")

		require.Error(t, err)
	})
}

func TestNewS3(t *testing.T) {
	t.Run("requires a bucket", func(t *testing.T) {
		_, err := NewS3(context.Background(), Config{Backend: "s3", Region: "us-east-1"})

		require.Error(t, err)
	})

	t.Run("requires a region", func(t *testing.T) {
		_, err := NewS3(context.Background(), Config{Backend: "s3", Bucket: "b"})

		require.Error(t, err)
	})

	t.Run("constructs with static credentials", func(t *testing.T) {
		s, err := NewS3(context.Background(), Config{
			Backend:   "s3",
			Bucket:    "b",
			Region:    "us-east-1",
			AccessKey: "ak",
			SecretKey: "sk",
			Endpoint:  "http://127.0.0.1:9000",
			PathStyle: true,
		})

		require.NoError(t, err)
		require.NotNil(t, s)
	})
}
