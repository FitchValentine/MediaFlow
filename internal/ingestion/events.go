package ingestion

import "time"

// IngestionEvent is emitted when raw media is accepted and stored.
type IngestionEvent struct {
	ID          string            `json:"id"`
	ObjectKey   string            `json:"object_key"`
	Checksum    string            `json:"checksum"`
	SizeBytes   int64             `json:"size_bytes"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
}
