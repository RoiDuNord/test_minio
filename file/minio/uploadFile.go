package minio

import (
	"context"
	"fmt"
	"log/slog"
	"s3_multiclient/load"
	"time"

	"github.com/minio/minio-go/v7"
)

const uploadChunkSize = 1024 * 1024 * 10

func (m *Minio) UploadFile(ctx context.Context, progressReader *load.ProgressReader, objectID, contentType, originalFileName string, contentLength int64) error {
	if err := m.uploadObjectToMinIO(ctx, progressReader, objectID, contentType, originalFileName, contentLength); err != nil {
		return err
	}

	return nil
}

func (m *Minio) uploadObjectToMinIO(ctx context.Context, progressReader *load.ProgressReader, objectID, contentType, originalFileName string, contentLength int64) error {
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		objectID,
		progressReader,
		contentLength,
		minio.PutObjectOptions{
			ContentType: contentType,
			PartSize:    uploadChunkSize,
			UserMetadata: map[string]string{
				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
				"X-Original-Name": originalFileName,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("ошибка при загрузке файла в MinIO: %v", err)
	}

	slog.Info("Медиафайл успешно загружен в MinIO", "object_id", objectID)
	return nil
}
