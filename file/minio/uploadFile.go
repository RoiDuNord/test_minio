package minio

import (
	"context"
	"fmt"
	"log/slog"
	"s3_multiclient/load"
	"s3_multiclient/server"
	"time"

	"github.com/minio/minio-go/v7"
)

const (
	uploadChunkSize = 5 * 1024 * 1024
	uploadedAtKey   = "X-Uploaded-At"
	originalNameKey = "X-Original-Name"
)

func (ml *MinioLoader) UploadFile(ctx context.Context, progressReader *load.ProgressReader, objectData *server.UploadRequestMetadata) error {
	_, err := ml.client.PutObject(
		ctx,
		ml.bucketName,
		objectData.ID,
		progressReader,
		objectData.Size,
		minio.PutObjectOptions{
			ContentType: objectData.ContentType,
			PartSize:    uploadChunkSize,
			UserMetadata: map[string]string{
				uploadedAtKey:   time.Now().Format(time.RFC3339),
				originalNameKey: objectData.FileName,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("ошибка при загрузке файла в MinIO: %v", err)
	}

	slog.Info("Медиафайл успешно загружен в MinIO", "object_id", objectData.ID)
	return nil
}

// Сообщения о статусе загрузки

// type FileDownloadResult struct {
//     RequestInfo      *DownloadRequest
//     StorageObject    S3ObjectHandle
//     Status           DownloadStatus
//     Error            error
// }

// type DownloadStatus int

// const (
//     Pending DownloadStatus = iota
//     InProgress
//     Completed
//     Failed
// )