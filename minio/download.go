package minio

import (
	"context"
	"log/slog"
	"net/http"
	"test_minio/file"
	"time"

	"github.com/minio/minio-go/v7"
)

// Download обрабатывает запрос на скачивание файла из MinIO
// @Summary Download content for an object
// @Description Downloads the content of the specified object ID from MinIO. Supports both regular files and ZIP archives.
// @Tags Objects
// @Produce application/octet-stream
// @Param object_id path string true "Object ID"
// @Success 200 "File downloaded successfully"
// @Failure 400 {object} string "Invalid request"
// @Failure 404 {object} string "Object not found"
// @Failure 500 {object} string "Internal server error"
// @Router /objects/{object_id}/content [get]

// func (m *Minio) DownloadFile(objectID string) (io.ReadCloser, error) {
// 	return nil, nil
// }

func (m *Minio) DownloadFile(w http.ResponseWriter, ctx context.Context, objectID string, crc32 uint32) error {
	startTime := time.Now()

	object, stat, err := m.getObjectAndStat(ctx, objectID)
	if err != nil {
		return err
	}

	if stat.ContentType == "application/zip" {
		if crc32 == 0 {
			slog.Error("Отсутствует CRC для ZIP", "object_id", objectID)
			// http.Error(w, "No CRC for ZIP", http.StatusBadRequest)
			return err
		}
		file.HandleZipFile(w, ctx, objectID, object, stat, crc32, stat.Size, startTime)
		return err
	}

	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID, "start_time", startTime)

	file.HandleRegularFile(w, ctx, objectID, object, stat, startTime)

	return nil
}

func (m *Minio) getObject(ctx context.Context, objectID string) (*minio.Object, error) {
	return m.client.GetObject(ctx, m.bucketName, objectID, minio.GetObjectOptions{})
	// return s.FileManager.Client.GetObject(ctx, s.FileManager.BucketName, objectID, minio.GetObjectOptions{})
}

func (m *Minio) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
	return m.client.StatObject(ctx, m.bucketName, objectID, minio.StatObjectOptions{})
}

func (m *Minio) getObjectAndStat(ctx context.Context, objectID string) (*minio.Object, minio.ObjectInfo, error) {
	object, err := m.getObject(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		// http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return &minio.Object{}, minio.ObjectInfo{}, err
	}

	stat, err := m.getObjectStat(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		// http.Error(w, "File not found", http.StatusNotFound)
		return &minio.Object{}, minio.ObjectInfo{}, err
	}

	return object, stat, nil
}

// func (m *Minio) determineFileName(stat minio.ObjectInfo) string {
// 	originalName := stat.UserMetadata["X-Original-Name"]
// 	if originalName != "" {
// 		return originalName
// 	}
// 	return stat.Key
// }

// func (m *Minio) handleRegularFile(w http.ResponseWriter, r *http.Request, objectID string, stat minio.ObjectInfo, startTime time.Time) {
// 	fileName := s.determineFileName(stat)
// 	downloadTime := time.Now()

// 	object, err := s.getObject(s.Ctx, objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 		return
// 	}
// 	defer object.Close()

// 	handler := FileHandler(handleFile)
// 	if err := handler(w, fileName, object, stat.ContentType); err != nil {
// 		slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("Файл отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
// }

// server := handler.NewServer(ctx, minio, cfg.MinIO.BucketName, cfg.App.Port)
// // cannot use minio (variable of struct type "test_minio/minio".Minio) as handler.FileManager value in argument to handler.NewServer: "test_minio/minio".Minio does not implement handler.FileManager (method DeleteFile has pointer receiver)

func (m *Minio) DeleteFile(objectID string) error {
	return nil
}
