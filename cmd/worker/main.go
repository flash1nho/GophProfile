package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/flash1nho/GophProfile/internal/config"
	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/internal/worker"
	"github.com/flash1nho/GophProfile/pkg/storage"
)

func main() {
	cfg := config.Load()

	db, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}

	s3Client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("s3 error: %v", err)
	}

	s3 := storage.New(s3Client, cfg.S3Bucket)

	repo := repository.NewAvatarRepository(db)
	w := worker.NewWorker(repo, s3)

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("rabbit connect error: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("rabbit channel error: %v", err)
	}

	msgs, err := ch.Consume("avatars.queue", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("consume error: %v", err)
	}

	log.Println("worker started")

	for msg := range msgs {
		if err := w.Handle(msg.Body); err != nil {
			log.Printf("worker handle error: %v", err)
		}
	}
}
