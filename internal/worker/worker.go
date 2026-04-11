package worker

import (
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/flash1nho/GophProfile/internal/repository"
	"github.com/flash1nho/GophProfile/pkg/storage"
)

type Worker struct {
	repo *repository.AvatarRepository
	s3   *storage.S3
	ch   *amqp.Channel
}

func NewWorker(repo *repository.AvatarRepository, s3 *storage.S3, ch *amqp.Channel) *Worker {
	return &Worker{
		repo: repo,
		s3:   s3,
		ch:   ch,
	}
}
