package minio

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"test_minio/handler/file"
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

type ProgressWriter struct {
	Writer io.Writer
	Total  int64
	Last   time.Time
}

func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{Writer: w, Last: time.Now()}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Total += int64(n)
	now := time.Now()
	if now.Sub(pw.Last) >= time.Second {
		slog.Info("Прогресс передачи данных", "total_bytes", pw.Total)
		pw.Last = now
	}
	return n, err
}

func (m *Minio) DownloadFile(w http.ResponseWriter, ctx context.Context, objectID string, crc32 uint32) error {
	startTime := time.Now()

	object, stat, err := m.getObjectAndStat(w, ctx, objectID)
	if err != nil {
		return err
	}

	if stat.ContentType == "application/zip" {
		if crc32 == 0 {
			slog.Error("Отсутствует CRC для ZIP", "object_id", objectID)
			http.Error(w, "No CRC for ZIP", http.StatusBadRequest)
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
}

func (m *Minio) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
	return m.client.StatObject(ctx, m.bucketName, objectID, minio.StatObjectOptions{})
}

func (m *Minio) getObjectAndStat(w http.ResponseWriter, ctx context.Context, objectID string) (*minio.Object, minio.ObjectInfo, error) {
	object, err := m.getObject(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return &minio.Object{}, minio.ObjectInfo{}, err
	}

	stat, err := m.getObjectStat(ctx, objectID)
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return &minio.Object{}, minio.ObjectInfo{}, err
	}

	return object, stat, nil
}
