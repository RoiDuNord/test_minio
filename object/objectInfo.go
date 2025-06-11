package object

import (
	"context"
	"log/slog"
	"strings"
	"test_minio/config"

	"github.com/minio/minio-go/v7"
)

func GetObjectInfo(ctx context.Context, minioClient *minio.Client, cfg *config.MinIOConfig, object *Object) error {
	objectMinio, err := minioClient.GetObject(ctx, cfg.BucketName, object.Name, minio.GetObjectOptions{})
	if err != nil {
		slog.Error("Ошибка при получении объекта", "bucketName", cfg.BucketName, "objectName", object.Name, "error", err)
		return err
	}
	defer objectMinio.Close()

	info, err := objectMinio.Stat()
	if err != nil {
		slog.Error("Ошибка при получении информации об объекте", "bucketName", cfg.BucketName, "objectName", object.Name, "error", err)
		return err
	}

	logObjectDetails(cfg, &info, object.Name)

	return nil
}

func logObjectDetails(cfg *config.MinIOConfig, objectInfo *minio.ObjectInfo, objectName string) {
	redObjectName := strings.TrimPrefix(objectName, "testdata/")
	slog.Info("Найден объект:", "bucketName", cfg.BucketName, "eTag", objectInfo.ETag, "objectName", redObjectName, "contentType", objectInfo.ContentType, "size", objectInfo.Size)
}
