package object

import (
	"context"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

func CreateBucket(ctx context.Context, minioClient *minio.Client, bucketName string, location string) error {
	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			slog.Info("Бакет уже существует", "bucketName", bucketName)
		} else {
			slog.Error("Ошибка при создании бакета", "error", err)
			return err
		}
	} else {
		slog.Info("Бакет успешно создан", "bucketName", bucketName)
	}
	return nil
}