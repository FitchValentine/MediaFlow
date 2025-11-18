package ingestion

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/your-org/mediaflow/pkg/kafka"
	"github.com/your-org/mediaflow/pkg/storage/objectstore"
)

// Service wires together storage, Kafka, and logging for ingestion flows.
type Service struct {
	store    objectstore.Client
	producer *kafka.Producer
	logger   *zap.Logger
}

type Params struct {
	Store    objectstore.Client
	Producer *kafka.Producer
	Logger   *zap.Logger
}

// UploadOptions captures metadata about the upload.
type UploadOptions struct {
	Filename    string
	ContentType string
	Metadata    map[string]string
}

type UploadResult struct {
	MediaID    string
	ObjectKey  string
	Checksum   string
	Size       int64
	UploadedAt time.Time
}

// NewService constructs an ingestion Service.
func NewService(p Params) *Service {
	return &Service{
		store:    p.Store,
		producer: p.Producer,
		logger:   p.Logger,
	}
}

// ProcessUpload streams the file to the object store and emits an event.
func (s *Service) ProcessUpload(ctx context.Context, reader io.Reader, size int64, opts UploadOptions) (*UploadResult, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid file size: %d", size)
	}

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)
	buffered := bufio.NewReaderSize(tee, 64*1024)

	objectKey := fmt.Sprintf("%s/%s", time.Now().UTC().Format("2006/01/02"), uuid.NewString())
	if opts.Filename != "" {
		objectKey = fmt.Sprintf("%s/%s", time.Now().UTC().Format("2006/01/02"), opts.Filename)
	}

	metadata := map[string]string{
		"original_filename": opts.Filename,
		"content_type":      opts.ContentType,
	}
	for k, v := range opts.Metadata {
		metadata[k] = v
	}

	if err := s.store.Put(ctx, objectKey, buffered, size, metadata); err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	mediaID := uuid.NewString()
	event := IngestionEvent{
		ID:          mediaID,
		ObjectKey:   objectKey,
		Checksum:    checksum,
		SizeBytes:   size,
		ContentType: opts.ContentType,
		Metadata:    metadata,
		CreatedAt:   time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal ingestion event: %w", err)
	}

	headers := map[string]string{
		"media_id":   mediaID,
		"event_type": "ingestion.created",
	}

	if err := s.producer.Publish(ctx, []byte(mediaID), payload, headers); err != nil {
		return nil, fmt.Errorf("publish ingestion event: %w", err)
	}

	return &UploadResult{
		MediaID:    mediaID,
		ObjectKey:  objectKey,
		Checksum:   checksum,
		Size:       size,
		UploadedAt: time.Now().UTC(),
	}, nil
}

// Close releases underlying resources.
func (s *Service) Close(ctx context.Context) error {
	if err := s.producer.Close(ctx); err != nil {
		return err
	}
	return s.store.Close()
}
