package minio

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"s3_multiclient/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	client     *minio.Client
	bucketName string
}

func Init(cfg config.MinIOConfig) (*Minio, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Region: cfg.Location,
		Secure: cfg.UseSSL,
	})
	if err != nil {
		slog.Error("Ошибка при подключении к MinIO", "error", err)
		return nil, err
	}

	slog.Info("minioClient подключен", "endpoint", cfg.Endpoint)
	return &Minio{
		client:     minioClient,
		bucketName: cfg.BucketName,
	}, nil
}

func (m *Minio) CreateBucket(ctx context.Context, location string) error {
	slog.Info("Попытка создания бакета")

	if err := m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{Region: location}); err != nil {
		if isBucketAlreadyExists(err) {
			slog.Info("Бакет уже существует")
			return nil
		}

		slog.Error("Ошибка создания бакета", "error", err)
		return fmt.Errorf("ошибка создания бакета: %w", err)
	}

	slog.Info("Бакет успешно создан")
	return nil
}

func isBucketAlreadyExists(err error) bool {
	errMsg := err.Error()
	return strings.Contains(errMsg, "BucketAlreadyExists") || strings.Contains(errMsg, "Your previous request to create the named bucket succeeded and you already own it.")
}
