package minio

import (
	"context"
	"fmt"
	"log/slog"
	"s3_multiclient/load"
	"s3_multiclient/server"

	"github.com/minio/minio-go/v7"
)

func (ml *MinioLoader) DownloadFile(ctx context.Context, pw *load.ProgressWriter, data *server.DownloadRequestMetadata) error {
	slog.Info("Начало обработки запроса на скачивание", "object_id", data.ID)

	minioObject, err := ml.getObjectAndMetadata(ctx, data.ID)
	if err != nil {
		return err
	}

	object := &downloadedFileData{
		metadata:    data,
		minioObject: *minioObject,
	}

	if object.minioObject.info.ContentType == "application/zip" {
		if object.metadata.CRC32 == 0 {
			return fmt.Errorf("отсутствует CRC для ZIP: object_id=%s", object.metadata.ID)
		}
		if err := streamFileFromZip(pw, object); err != nil {
			return fmt.Errorf("ошибка при обработке ZIP-файла: %w", err)
		}
		return nil
	}

	if err := streamRegularFile(pw, object); err != nil {
		return fmt.Errorf("ошибка при обработке обычного файла: %w", err)
	}

	return nil
}

func (ml *MinioLoader) getObjectAndMetadata(ctx context.Context, objectID string) (*minioFileObject, error) {
	content, err := ml.client.GetObject(ctx, ml.bucketName, objectID, minio.GetObjectOptions{})
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		return nil, fmt.Errorf("не удалось получить объект: %w", err)
	}

	stat, err := ml.client.StatObject(ctx, ml.bucketName, objectID, minio.StatObjectOptions{})
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		return nil, fmt.Errorf("не удалось получить метаданные объекта: %w", err)
	}

	object := &minioFileObject{
		reader: content,
		info:   stat,
	}

	return object, nil
}

// Сообщения о статусе скачивания

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

// type DownloadRequest struct {
//     FileID        string
//     CRC32Checksum uint32
// }

// type FileDownloadResult struct {
//     RequestInfo      *DownloadRequest
//     StorageObject    S3ObjectHandle
//     DownloadStatus   string // или enum для статуса
//     DownloadError    error  // для хранения ошибок скачивания
// }

// type S3ObjectHandle struct {
//     ObjectReader   *minio.Object
//     ObjectMetadata minio.ObjectInfo
// }
