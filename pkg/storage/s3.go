package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/flash1nho/GophProfile/pkg/utils"
	"github.com/minio/minio-go/v7"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type S3 struct {
	client *minio.Client
	bucket string
	cb     *gobreaker.CircuitBreaker
}

func New(client *minio.Client, bucket string) *S3 {
	return &S3{client: client, bucket: bucket, cb: utils.NewCircuitBreaker("s3")}
}

func (s *S3) Upload(ctx context.Context, key string, data []byte, mime string) error {
	ctx, span := otel.Tracer("s3").Start(ctx, "PutObject")
	defer span.End()

	span.SetAttributes(
		attribute.String("bucket", s.bucket),
	)

	_, err := s.cb.Execute(func() (interface{}, error) {
		return s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: mime})
	})

	return err
}

func (s *S3) Download(ctx context.Context, key string) ([]byte, error) {
	res, err := s.cb.Execute(func() (interface{}, error) {
		return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	})

	if err != nil {
		return nil, err
	}

	obj := res.(*minio.Object)
	defer obj.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *S3) Health(ctx context.Context) error {
	res, err := s.cb.Execute(func() (interface{}, error) {
		return s.client.BucketExists(ctx, s.bucket)
	})

	if err != nil {
		return err
	}

	exists := res.(bool)
	if !exists {
		return fmt.Errorf("bucket does not exist")
	}
	return nil
}

func (s *S3) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.cb.Execute(func() (interface{}, error) {
		return s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	})

	if err != nil {
		errResp, ok := err.(minio.ErrorResponse)
		if ok && errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.cb.Execute(func() (interface{}, error) {
		return nil, s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	})

	return err
}
