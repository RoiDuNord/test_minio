package buckets

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/minio/minio-go/v7"
)

func Create(ctx context.Context, client *minio.Client, bucketName string, location string) error {
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		slog.Error("Ошибка проверки существования бакета", "error", err)
		return fmt.Errorf("ошибка проверки существования бакета: %w", err)
	}
	if exists {
		slog.Warn("Бакет уже существует, пропускаем создание")
		return nil
	}

	slog.Info("Попытка создания бакета")
	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
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
