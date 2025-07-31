package minio

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"test_minio/config"

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
	})
	if err != nil {
		slog.Error("Ошибка при подключении к MinIO", "error", err)
		return nil, err
	}

	slog.Info("minioClient подключен", "endpoint", cfg.Endpoint)
	return &Minio{
		client: minioClient,
	}, nil
}

func (m *Minio) CreateBucket(ctx context.Context, bucketName string, location string) error {
	slog.Info("Попытка создания бакета")
	err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		if isBucketAlreadyExists(err) {
			slog.Info("Бакет был создан параллельно другим процессом")
			return nil
		}

		slog.Error("Ошибка создания бакета", "error", err)
		return fmt.Errorf("ошибка создания бакета:%w", err)
	}

	slog.Info("Бакет успешно создан")
	return nil
}

func isBucketAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "BucketAlreadyExists")
}

// func (m *Minio) GetObject(ctx, objID) {
// 	m.client.GetObject(ctx, m.bucketName, objID, minio.GetObjectOptions{})
// }
