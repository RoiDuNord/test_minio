package minio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

const (
	statusCreated   = "created"
	uploadMessage   = "файл успешно загружен"
	defaultFileName = "default_name.bin"
	uploadChunkSize = 1024 * 1024 * 10
)

type ObjectResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	Message        string  `json:"message"`
	Duration       float64 `json:"duration_sec"`
	UploadDuration float64 `json:"uploadDuration_sec"`
}

func (m *Minio) UploadFile(objectID string, data io.Reader) error {
	originalName, err := parseFileNameFromDisposition(r)
	if err != nil {
		slog.Warn("Не удалось извлечь имя файла", "error", err)
	}

	objectID, err := parseObjectID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Не удалось извлечь object_id", "error", err)
		return
	}

	contentType := getContentType(originalName)
	slog.Info("Определен тип содержимого файла", "file_name", originalName, "content_type", contentType)

	uploadStart := time.Now()

	progressReader := &ProgressReader{
		r:    r.Body,
		last: time.Now(),
	}

	if err := s.uploadObjectToMinIO(objectID, contentType, originalName, progressReader, r.ContentLength); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Ошибка загрузки медиафайла в MinIO", "object_id", objectID, "error", err)
		return
	}

	if originalName == defaultFileName {
		originalName = generateFileName(contentType)
	}

	sendJSONResponse(w, objectID, originalName, contentType, startTime, uploadStart)
	return nil
}

type ProgressReader struct {
	r          io.Reader
	totalBytes int
	chunkCount int
	last       time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if err != nil {
		return n, err
	}
	pr.totalBytes += n
	pr.chunkCount++
	now := time.Now()
	if time.Since(pr.last) >= 1*time.Second {
		slog.Info("Прогресс чтения", "chunk_number", pr.chunkCount, "bytes_write_in_chunk", n, "total_Mb", pr.totalBytes/1024/1024)
		pr.last = now
	}
	return n, err
}

func checkPostMethod(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
	return nil
}

func parseObjectID(r *http.Request) (string, error) {
	objectID := chi.URLParam(r, "object_id")
	if objectID == "" {
		return "", fmt.Errorf("object_id is required")
	}
	return objectID, nil
}

func parseFileNameFromDisposition(r *http.Request) (string, error) {
	originalName := defaultFileName
	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				name = filepath.Base(name)
				name = strings.ReplaceAll(name, "..", "")
				if name != "" {
					originalName = name
					return originalName, nil
				}
			}
		} else {
			slog.Warn("Ошибка разбора Content-Disposition", "error", err)
		}
	}
	return originalName, nil
}

func generateFileName(contentType string) string {
	originalName := defaultFileName
	ext := filepath.Ext(originalName)
	if ext == "" {
		exts, err := mime.ExtensionsByType(contentType)
		if err != nil {
			slog.Warn("Не удалось определить расширение по Content-Type", "content_type", contentType, "error", err)
			ext = ".bin"
		} else if len(exts) > 0 {
			ext = exts[0]
		} else {
			ext = ".bin"
		}
		originalName = uuid.NewString()
	}
	return originalName
}

func (m *Minio) uploadObjectToMinIO(ctx context.Context, objectID, contentType, originalName string, body io.Reader, size int64) error {
	uploadStart := time.Now()
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		objectID,
		body,
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
	slog.Info("Медиафайл успешно загружен в MinIO", "object_id", objectID, "upload_duration", time.Since(uploadStart).Seconds())
	return nil
}

func sendJSONResponse(w http.ResponseWriter, objectID, originalName, contentType string, startTime, uploadStart time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	duration := time.Since(startTime)
	uploadDuration := time.Since(uploadStart)

	response := &ObjectResponse{
		ID:             objectID,
		Name:           originalName,
		Type:           contentType,
		Status:         statusCreated,
		Message:        uploadMessage,
		Duration:       duration.Seconds(),
		UploadDuration: uploadDuration.Seconds(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Ошибка формирования JSON ответа", "error", err)
	}
}
