package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/flash1nho/GophProfile/internal/api"
	"github.com/flash1nho/GophProfile/internal/config"
	"github.com/flash1nho/GophProfile/internal/handlers"
	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/internal/services"
	"github.com/flash1nho/GophProfile/pkg/broker"
	"github.com/flash1nho/GophProfile/pkg/storage"
)

func main() {
	cfg := config.Load()

	// DB
	db, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	// S3
	s3Client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("s3 init error: %v", err)
	}

	s3 := storage.New(s3Client, cfg.S3Bucket)

	// RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("rabbit connect error: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("rabbit channel error: %v", err)
	}

	rabbit, err := broker.New(ch)
	if err != nil {
		log.Fatalf("rabbit init error: %v", err)
	}

	// Layers
	repo := repository.NewAvatarRepository(db)
	service := services.NewAvatarService(repo, s3, rabbit)
	handler := handlers.NewAvatarHandler(service)

	router := api.NewRouter(handler)

	log.Println("server started :8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
