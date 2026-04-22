package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/flash1nho/GophProfile/internal/config"
	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/internal/worker"
	"github.com/flash1nho/GophProfile/pkg/logger"
	"github.com/flash1nho/GophProfile/pkg/storage"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	"github.com/flash1nho/GophProfile/internal/observability"
)

func main() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := log.Sync(); err != nil {
			log.Error("failed to sync logger", zap.Error(err))
		}
	}()

	log.Info("starting worker")

	cfg := config.New(log)

	shutdownTracer := observability.InitTracer("worker")
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			log.Error("tracer shutdown failed", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatal("db error", zap.Error(err))
	}
	defer func() {
		log.Info("closing db")
		db.Close()
	}()

	s3Client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("s3 error", zap.Error(err))
	}

	s3 := storage.New(s3Client, cfg.S3Bucket)

	repo := repository.NewAvatarRepository(db, log)
	w := worker.NewWorker(repo, s3)

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatal("rabbit connect error", zap.Error(err))
	}
	defer func() {
		log.Info("closing rabbit connection")
		if err := conn.Close(); err != nil {
			log.Error("failed to close rabbit connection", zap.Error(err))
		}
	}()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("rabbit channel error", zap.Error(err))
	}
	defer func() {
		log.Info("closing rabbit channel")
		if err := ch.Close(); err != nil {
			log.Error("failed to close rabbit channel", zap.Error(err))
		}
	}()

	msgs, err := ch.Consume(
		"avatars.queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("consume error", zap.Error(err))
	}

	log.Info("worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		<-ctx.Done()
		log.Info("closing rabbit channel (shutdown)")
		if err := ch.Close(); err != nil {
			log.Error("failed to close channel on shutdown", zap.Error(err))
		}
	}()

	go func() {
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				log.Info("worker context cancelled")
				return

			case msg, ok := <-msgs:
				if !ok {
					log.Warn("rabbit channel closed")
					return
				}

				carrier := propagation.MapCarrier{}

				if msg.Headers != nil {
					for k, v := range msg.Headers {
						if str, ok := v.(string); ok {
							carrier[k] = str
						}
					}
				}

				ctxMsg := otel.GetTextMapPropagator().Extract(ctx, carrier)
				ctxMsg, span := otel.Tracer("worker").Start(ctxMsg, "rabbit.consume.upload_event")

				span.SetAttributes(
					attribute.String("messaging.system", "rabbitmq"),
					attribute.String("messaging.destination", "avatars.queue"),
					attribute.String("messaging.message_id", msg.MessageId),
					attribute.Int("messaging.body_size", len(msg.Body)),
				)

				logger := observability.WithTrace(ctxMsg, log)

				logger.Info("message received",
					zap.Int("body_size", len(msg.Body)),
					zap.String("message_id", msg.MessageId),
				)

				if err := w.HandleUploadEvent(ctxMsg, msg.Body); err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					logger.Error("worker error", zap.Error(err))

					if err := msg.Nack(false, true); err != nil {
						logger.Error("failed to nack message", zap.Error(err))
					}
				} else {
					if err := msg.Ack(false); err != nil {
						logger.Error("failed to ack message", zap.Error(err))
					}
				}

				span.End()
			}
		}
	}()

	sig := <-quit
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	cancel()

	select {
	case <-done:
		log.Info("worker stopped gracefully")

	case <-time.After(5 * time.Second):
		log.Warn("worker shutdown timeout")
	}

	log.Info("worker exited")
}
