package main

import (
	"context"

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
)

func main() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting worker")

	cfg := config.Load(log)

	db, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal("db error", zap.Error(err))
	}

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

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("rabbit channel error", zap.Error(err))
	}

	msgs, err := ch.Consume("avatars.queue", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("consume error", zap.Error(err))
	}

	log.Info("worker started")

	for msg := range msgs {
		if err := w.HandleUploadEvent(msg.Body); err != nil {
			log.Fatal("worker error", zap.Error(err))
		}
	}
}
