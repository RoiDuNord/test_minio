package minio

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"test_minio/models"
	"time"

	"github.com/minio/minio-go/v7"
)

const uploadChunkSize = 1024 * 1024 * 10

type ObjectResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	Message        string  `json:"message"`
	Duration       float64 `json:"duration_sec"`
	UploadDuration float64 `json:"uploadDuration_sec"`
}

func (m *Minio) UploadFile(r *http.Request, ctx context.Context, objectID, contentType, originalFileName string) error {
	progressReader := &models.ProgressReader{
		Reader:      r.Body,
		LastLogTime: time.Now(),
	}

	if err := m.uploadObjectToMinIO(ctx, objectID, contentType, originalFileName, progressReader, r.ContentLength); err != nil {
		return err
	}

	return nil
}

func (m *Minio) uploadObjectToMinIO(ctx context.Context, objectID, contentType, originalName string, progressReader *models.ProgressReader, size int64) error {
	startTime := time.Now()

	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		objectID,
		progressReader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
			PartSize:    uploadChunkSize,
			UserMetadata: map[string]string{
				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
				"X-Original-Name": originalName,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upload media to MinIO: %v", err)
	}

	uploadDuration := time.Since(startTime).Seconds()
	slog.Info("Медиафайл успешно загружен в MinIO", "object_id", objectID, "upload_duration", uploadDuration)
	return nil
}
