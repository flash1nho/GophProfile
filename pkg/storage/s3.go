package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func New(client *minio.Client, bucket string) *S3 {
	return &S3{client: client, bucket: bucket}
}

func (s *S3) Upload(ctx context.Context, key string, data []byte, mime string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: mime})
	return err
}

func (s *S3) Download(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *S3) Health(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket does not exist")
	}
	return nil
}
