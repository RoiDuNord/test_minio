package handler

import (
	"log/slog"
	"net/http"
)

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

	contentType := getContentType(originalFileName)
	slog.Info("Определен тип содержимого файла", "file_name", originalFileName, "content_type", contentType)

	if err := s.loadManager.Upload(r, s.ctx, objectID, contentType, originalFileName, r.ContentLength); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, objectID, originalFileName, contentType)
}
