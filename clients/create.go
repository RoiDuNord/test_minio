package clients

import (
	"log/slog"
	"test_minio/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Create(cfg config.MinIOConfig) (*minio.Client, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Location,
	})
	if err != nil {
		slog.Error("Ошибка при подключении к MinIO", "error", err)
		return nil, err
	}	

	slog.Info("minioClient подключен", "endpoint", cfg.Endpoint)
	return minioClient, nil
}
