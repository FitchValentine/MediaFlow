package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/your-org/mediaflow/internal/ingestion"
	"github.com/your-org/mediaflow/pkg/config"
	"github.com/your-org/mediaflow/pkg/kafka"
	"github.com/your-org/mediaflow/pkg/logger"
	"github.com/your-org/mediaflow/pkg/storage/objectstore"
	"github.com/your-org/mediaflow/pkg/tracing"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logr, err := logger.New(cfg.App.LogLevel)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logr.Sync() //nolint:errcheck

	traceShutdown, err := tracing.Init(ctx, tracing.Config{
		Endpoint:    cfg.Tracing.Endpoint,
		Insecure:    cfg.Tracing.Insecure,
		SampleRatio: cfg.Tracing.SampleRatio,
		Attributes:  parseResourceAttributes(cfg.Tracing.ResourceAttr),
		ServiceName: cfg.App.Name,
	})
	if err != nil {
		logr.Fatal("init tracing", zap.Error(err))
	}
	defer traceShutdown(context.Background()) //nolint:errcheck

	producer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers:      cfg.Kafka.Brokers,
		Topic:        cfg.Kafka.IngestionTopic,
		BatchSize:    cfg.Kafka.BatchSize,
		BatchTimeout: cfg.Kafka.BatchTimeout,
		Compression:  kafka.CompressionFromString(cfg.Kafka.CompressionCodec),
		RequiredAcks: kafkago.RequireAll,
		MaxAttempts:  cfg.Kafka.Retries,
	})

	store, err := objectstore.New(objectstore.Config{
		Provider:  cfg.Storage.Provider,
		Endpoint:  cfg.Storage.Endpoint,
		Region:    cfg.Storage.Region,
		Bucket:    cfg.Storage.Bucket,
		AccessKey: cfg.Storage.AccessKey,
		SecretKey: cfg.Storage.SecretKey,
		UseSSL:    cfg.Storage.UseSSL,
	})
	if err != nil {
		logr.Fatal("init object store", zap.Error(err))
	}

	service := ingestion.NewService(ingestion.Params{
		Store:    store,
		Producer: producer,
		Logger:   logr,
	})

	handler := ingestion.NewHTTPHandler(service, logr, cfg.Upload.MaxSizeBytes, cfg.Upload.MultipartMemBytes)

	server := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler.Router(),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logr.Error("http server shutdown failed", zap.Error(err))
		}
		if err := service.Close(shutdownCtx); err != nil {
			logr.Error("service shutdown failed", zap.Error(err))
		}
	}()

	logr.Info("ingestion service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logr.Fatal("http server failed", zap.Error(err))
	}
}

func parseResourceAttributes(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	attrs := map[string]string{}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		if !strings.Contains(pair, "=") {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		attrs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return attrs
}
