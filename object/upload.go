package object

import (
	"context"
	"log/slog"
	"test_minio/config"

	"github.com/minio/minio-go/v7"
)

func UploadObject(ctx context.Context, minioClient *minio.Client, cfg *config.MinIOConfig, objectInfo *Object) error {
	info, err := minioClient.FPutObject(ctx, cfg.BucketName, objectInfo.Name, objectInfo.Path, minio.PutObjectOptions{ContentType: objectInfo.ContentType})
	if err != nil {
		slog.Error("Ошибка при загрузке файла в MinIO", "bucketName", cfg.BucketName, "objectName", objectInfo.Name, "objectPath", objectInfo.Path, "error", err)
		return err
	}

	slog.Info("Файл успешно загружен", "bucketName", cfg.BucketName, "objectName", objectInfo.Name, "size", info.Size)
	return nil
}
