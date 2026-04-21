package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/flash1nho/GophProfile/internal/observability"
	"github.com/flash1nho/GophProfile/pkg/broker"
	"github.com/flash1nho/GophProfile/pkg/cache"
	"github.com/flash1nho/GophProfile/pkg/logger"
	"github.com/flash1nho/GophProfile/pkg/storage"
)

func main() {
	log, err := logger.New()
	log = observability.WithTrace(ctx, log)

	if err != nil {
		panic(err)
	}

	defer func() {
		if err := log.Sync(); err != nil {
			log.Error("failed to sync logger", zap.Error(err))
		}
	}()

	log.Info("starting http server")

	ctx := context.Background()

	shutdownTracer := observability.InitTracer("gophprofile")
	defer func() {
	    if err := shutdownTracer(ctx); err != nil {
	        log.Error("tracer shutdown failed", zap.Error(err))
	    }
	}()

	cfg := config.New(log)

	db, err := observability.NewPGXPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatal("db connect error", zap.Error(err))
	}
	defer func() {
		log.Info("closing db")
		db.Close()
	}()

	s3Client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
		Secure: false,
	})
	s3Client = observability.WrapMinioClient(s3Client)
	if err != nil {
		log.Fatal("s3 init error", zap.Error(err))
	}

	s3 := storage.New(s3Client, cfg.S3Bucket)

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatal("rabbit connect error", zap.Error(err))
	}
	defer func() {
		log.Info("closing rabbit connection")
		conn.Close()
	}()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("rabbit channel error", zap.Error(err))
	}
	defer func() {
		log.Info("closing rabbit channel")
		ch.Close()
	}()

	rabbit, err := broker.New(conn, ch)
	rabbit = observability.WrapBroker(rabbit)
	if err != nil {
		log.Fatal("rabbit init error", zap.Error(err))
	}

	repo := repository.NewAvatarRepository(db, log)
	service := services.NewAvatarService(repo, s3, rabbit, log)

	cache := cache.NewMemoryCache[[]byte]()
	handler := handlers.NewAvatarHandler(service, log, cache)

	router := api.NewRouter(handler, log)
	addr := ":8080"

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Info("http server started", zap.String("addr", addr))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("server shutdown failed", zap.Error(err))
	}

	log.Info("server exited properly")
}
