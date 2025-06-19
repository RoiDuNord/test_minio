package handler

import (
	"bytes"
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

// Upload handles the file upload for a specific object.
//
// @Summary Upload content for an object
// @Description Uploads a file or content for the specified object ID. Supports various file types including ZIP archives.
// @Tags Objects
// @Accept application/octet-stream
// @Produce json
// @Param object_id path string true "Object ID"
// @Param file formData file true "File to upload" format(binary)
// @Success 201 {object} ObjectResponse "Successful upload response"
// @Failure 400 {object} ObjectResponse "Invalid request or file type"
// @Failure 405 {object} ObjectResponse "Method not allowed"
// @Failure 500 {object} ObjectResponse "Internal server error"
// @Router /objects/{object_id}/content [post]
func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	slog.Info("Начало обработки запроса на загрузку")

	if err := checkRequestMethod(r); err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса", "error", err)
		return
	}

	originalName, err := extractFileName(r)
	if err != nil {
		slog.Warn("Не удалось извлечь имя файла", "error", err)
	}

	objectID, err := getObjectID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Не удалось извлечь object_id", "error", err)
	}
	slog.Info("Object_id успешно получен", "object_id", objectID)

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		buf := make([]byte, 512)
		n, err := io.ReadAtLeast(r.Body, buf, 1)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			slog.Error("Ошибка чтения тела запроса", "error", err)
			return
		}
		contentType = http.DetectContentType(buf[:n])
		slog.Info("Определён Content-Type", "content_type", contentType)
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf[:n]), r.Body))
	}

	uploadStart := time.Now()

	if isStreamContent(contentType) {
		// Для медиафайлов и ZIP используем потоковую загрузку
		if err := s.uploadMediaToMinIO(objectID, contentType, originalName, r.Body, -1); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Ошибка загрузки медиафайла в MinIO", "object_id", objectID, "error", err)
			return
		}
	} else {
		// Для документов и изображений читаем данные в память
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			slog.Error("Ошибка чтения тела запроса", "error", err)
			return
		}
		if err := s.uploadDocumentToMinIO(objectID, contentType, originalName, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Ошибка загрузки документа в MinIO", "object_id", objectID, "error", err)
			return
		}
	}

	if originalName == defaultFileName {
		originalName = generateFileName(contentType)
	}

	sendJSONResponse(w, objectID, originalName, contentType, startTime, uploadStart)
}

const (
	statusCreated   = "created"
	uploadMessage   = "файл успешно загружен"
	defaultFileName = "default_name.bin"
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

func checkRequestMethod(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
	return nil
}

func getObjectID(r *http.Request) (string, error) {
	objectID := chi.URLParam(r, "object_id")
	if objectID == "" {
		return "", fmt.Errorf("object_id is required")
	}
	return objectID, nil
}

// extractFileName извлекает имя файла из Content-Disposition
func extractFileName(r *http.Request) (string, error) {
	originalName := defaultFileName
	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				name = filepath.Base(name)
				name = strings.ReplaceAll(name, "..", "")
				if name != "" {
					originalName = name
					slog.Info("Имя файла извлечено из Content-Disposition", "fileName", originalName)
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
		slog.Info("Сгенерировано имя файла", "fileName", originalName)
	}
	return originalName
}

func isStreamContent(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/") ||
		strings.HasPrefix(contentType, "application/zip") ||
		strings.HasPrefix(contentType, "application/x-zip-compressed")
}

func (s *Server) uploadDocumentToMinIO(objectID, contentType, originalName string, data []byte) error {
	uploadStart := time.Now()
	_, err := s.MinioClient.Client.PutObject(
		s.Ctx,
		s.MinioClient.BucketName,
		objectID,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
				"X-Original-Name": originalName,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upload document to MinIO: %v", err)
	}
	slog.Info("Документ успешно загружен в MinIO", "object_id", objectID, "upload_duration", time.Since(uploadStart).Seconds())
	return nil
}

func (s *Server) uploadMediaToMinIO(objectID, contentType, originalName string, body io.Reader, size int64) error {
	uploadStart := time.Now()
	_, err := s.MinioClient.Client.PutObject(
		s.Ctx,
		s.MinioClient.BucketName,
		objectID,
		body,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
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
