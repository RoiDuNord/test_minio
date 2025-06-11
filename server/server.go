package server

import (
	"context"
	"log/slog"
	"test_minio/client"
	"test_minio/config"
	"test_minio/object"
)

func Run() {
	cfg, err := config.Get()
	if err != nil {
		slog.Error("Ошибка при загрузке конфигурации из .env", "error", err)
		return
	}
	slog.Info("config успешно загружен")

	minioClient, err := client.Get(*cfg)
	if err != nil {
		slog.Error("Ошибка при подключении minioClient", "error", err)
		return
	}
	slog.Info("minioClient подключен", "endpoint", cfg.Endpoint)

	ctx := context.Background()
	err = object.CreateBucket(ctx, minioClient, cfg.BucketName, cfg.Location)
	if err != nil {
		slog.Error("Ошибка при создании бакета", "error", err)
		return
	}

	objectInfo, err := object.Get()
	if err != nil {
		slog.Error("Ошибка при получении информации об объекте", "error", err)
		return
	}

	err = object.UploadObject(ctx, minioClient, cfg, objectInfo)
	if err != nil {
		slog.Error("Ошибка при загрузке объекта", "error", err)
		return
	}

	err = object.GetObjectInfo(ctx, minioClient, cfg, objectInfo)
	if err != nil {
		slog.Error("Ошибка при получении информации об объекте из MinIO", "error", err)
		return
	}
}
