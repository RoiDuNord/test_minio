package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"test_minio/handler/file"

	"github.com/go-chi/chi"
)

const successfulUploadStatus = "uploaded"

type ObjectResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	slog.Info("Начало обработки запроса на загрузку")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	originalFileName, err := parseFileNameFromDisposition(r)
	if err != nil {
		slog.Warn("Не удалось извлечь имя файла", "error", err)
	}

	objectID, err := parseObjectID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Не удалось извлечь object_id", "error", err)
		return
	}

	contentType := file.GetContentType(originalFileName)
	slog.Info("Определен тип содержимого файла", "file_name", originalFileName, "content_type", contentType)

	if err := s.FileManager.UploadFile(r, s.Ctx, objectID, contentType, originalFileName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Ошибка загрузки медиафайла в MinIO", "object_id", objectID, "error", err)
		return
	}

	sendJSONResponse(w, objectID, originalFileName, contentType)
}

func parseObjectID(r *http.Request) (string, error) {
	objectID := chi.URLParam(r, "object_id")
	if objectID == "" {
		return "", fmt.Errorf("object_id is required")
	}
	return objectID, nil
}

func parseFileNameFromDisposition(r *http.Request) (string, error) {
	originalName := file.DefaultUploadFileName
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

func sendJSONResponse(w http.ResponseWriter, objectID, originalName, contentType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := &ObjectResponse{
		ID:     objectID,
		Name:   originalName,
		Type:   contentType,
		Status: successfulUploadStatus,
		// Message: successfulUploadMessage,
		// UploadDuration: uploadDuration.Seconds(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Ошибка формирования JSON ответа", "error", err)
	}
}
