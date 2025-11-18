package objectstore

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config contains the information required to talk to an object store.
type Config struct {
	Provider  string
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// Client represents the capabilities the ingestion service expects.
type Client interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, metadata map[string]string) error
	Close() error
}

// New creates an object store client based on the given configuration.
func New(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "minio", "s3":
		return newMinioClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported object store provider: %s", cfg.Provider)
	}
}

type minioClient struct {
	client *minio.Client
	bucket string
}

func newMinioClient(cfg Config) (Client, error) {
	cl, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	return &minioClient{client: cl, bucket: cfg.Bucket}, nil
}

func (m *minioClient) Put(ctx context.Context, key string, reader io.Reader, size int64, metadata map[string]string) error {
	opts := minio.PutObjectOptions{UserMetadata: metadata}
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, opts)
	return err
}

func (m *minioClient) Close() error {
	return nil
}
