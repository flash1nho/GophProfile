package main

import (
	"context"
	"net/http"

	"go.uber.org/zap"

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
	"github.com/flash1nho/GophProfile/pkg/logger"
	"github.com/flash1nho/GophProfile/pkg/storage"
)

func main() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting server")

	cfg := config.Load(log)

	db, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Error("db connect error", zap.Error(err))
	}

	s3Client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("s3 init error", zap.Error(err))
	}

	s3 := storage.New(s3Client, cfg.S3Bucket)

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatal("rabbit connect error", zap.Error(err))
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("rabbit channel error", zap.Error(err))
	}

	rabbit, err := broker.New(ch)
	if err != nil {
		log.Fatal("rabbit init error", zap.Error(err))
	}

	repo := repository.NewAvatarRepository(db, log)
	service := services.NewAvatarService(repo, s3, rabbit, log)
	handler := handlers.NewAvatarHandler(service)

	router := api.NewRouter(handler, log)

	log.Info("server started :8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
