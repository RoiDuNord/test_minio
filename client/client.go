package client

import (
	"log/slog"
	"test_minio/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Get(cfg config.MinIOConfig) (*minio.Client, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		slog.Error("Ошибка при подключении к MinIO", "error", err)
		return nil, err
	}

	return minioClient, nil
}
