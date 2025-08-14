package minio

import (
	"context"
	"fmt"
	"log/slog"
	"s3_multiclient/load"

	"github.com/minio/minio-go/v7"
)

func (m *Minio) DownloadFile(ctx context.Context, pw *load.ProgressWriter, objectID string, crc32 uint32) error {
	object, stat, err := m.getObjectAndStat(ctx, objectID)
	if err != nil {
		return err
	}

	if stat.ContentType == "application/zip" {
		if crc32 == 0 {
			slog.Error("Отсутствует CRC для ZIP", "object_id", objectID)
			return fmt.Errorf("отсутствует CRC для ZIP: object_id=%s", objectID)
		}
		if err := handleZipFile(pw, objectID, object, crc32, stat.Size); err != nil {
			return fmt.Errorf("ошибка при обработке ZIP-файла: %w", err)
		}
		return nil
	}

	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID)

	if err := handleRegularFile(pw, objectID, object, stat); err != nil {
		return fmt.Errorf("ошибка при обработке обычного файла: %w", err)
	}

	return nil
}

func (m *Minio) getObject(ctx context.Context, objectID string) (*minio.Object, error) {
	return m.client.GetObject(ctx, m.bucketName, objectID, minio.GetObjectOptions{})
}

func (m *Minio) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
	return m.client.StatObject(ctx, m.bucketName, objectID, minio.StatObjectOptions{})
}

func (m *Minio) getObjectAndStat(ctx context.Context, objectID string) (*minio.Object, minio.ObjectInfo, error) {
	object, err := m.getObject(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		return nil, minio.ObjectInfo{}, fmt.Errorf("не удалось получить объект: %w", err)
	}

	stat, err := m.getObjectStat(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		return nil, minio.ObjectInfo{}, fmt.Errorf("не удалось получить метаданные объекта: %w", err)
	}

	return object, stat, nil
}
