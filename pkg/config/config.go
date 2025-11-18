package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// Config captures the full runtime configuration for a MediaFlow service.
type Config struct {
	App     AppConfig
	HTTP    HTTPConfig
	GRPC    GRPCConfig
	Kafka   KafkaConfig
	Storage StorageConfig
	Tracing TracingConfig
	Metrics MetricsConfig
	Upload  UploadConfig
}

type AppConfig struct {
	Name        string `env:"APP_NAME" envDefault:"mediaflow-ingestion"`
	Environment string `env:"APP_ENV" envDefault:"development"`
	Version     string `env:"APP_VERSION" envDefault:"0.1.0"`
	LogLevel    string `env:"APP_LOG_LEVEL" envDefault:"info"`
}

type HTTPConfig struct {
	Addr         string        `env:"HTTP_ADDR" envDefault:":8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"60s"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"120s"`
}

type GRPCConfig struct {
	Addr string `env:"GRPC_ADDR" envDefault:":9090"`
}

type KafkaConfig struct {
	Brokers          []string      `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
	IngestionTopic   string        `env:"KAFKA_INGESTION_TOPIC" envDefault:"mediaflow.ingestion"`
	Retries          int           `env:"KAFKA_RETRIES" envDefault:"3"`
	RetryBackoff     time.Duration `env:"KAFKA_RETRY_BACKOFF" envDefault:"500ms"`
	CompressionCodec string        `env:"KAFKA_COMPRESSION_CODEC" envDefault:"snappy"`
	BatchSize        int           `env:"KAFKA_BATCH_SIZE" envDefault:"100"`
	BatchTimeout     time.Duration `env:"KAFKA_BATCH_TIMEOUT" envDefault:"1s"`
}

type StorageConfig struct {
	Provider  string `env:"STORAGE_PROVIDER" envDefault:"minio"`
	Endpoint  string `env:"STORAGE_ENDPOINT" envDefault:"http://localhost:9000"`
	Region    string `env:"STORAGE_REGION" envDefault:"us-east-1"`
	Bucket    string `env:"STORAGE_BUCKET" envDefault:"mediaflow-raw"`
	AccessKey string `env:"STORAGE_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"STORAGE_SECRET_KEY" envDefault:"minioadmin"`
	UseSSL    bool   `env:"STORAGE_USE_SSL" envDefault:"false"`
}

type TracingConfig struct {
	Endpoint     string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	Insecure     bool    `env:"OTEL_EXPORTER_OTLP_INSECURE" envDefault:"true"`
	SampleRatio  float64 `env:"OTEL_TRACES_SAMPLER_RATIO" envDefault:"1.0"`
	ResourceAttr string  `env:"OTEL_RESOURCE_ATTRIBUTES" envDefault:"service.namespace=mediaflow"`
}

type MetricsConfig struct {
	Addr string `env:"METRICS_ADDR" envDefault:":9102"`
}

type UploadConfig struct {
	MaxSizeBytes      int64 `env:"UPLOAD_MAX_SIZE_BYTES" envDefault:"10737418240"`
	MultipartMemBytes int64 `env:"UPLOAD_MULTIPART_MEM_BYTES" envDefault:"52428800"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
