package handler

import (
	"log/slog"
	"net/http"
	"test_minio/file"

	"github.com/go-chi/chi"
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
func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	parsedInfo := chi.URLParam(r, "object_id")
	objectID, crc32, err := file.ParseObjectIDandCRC(parsedInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.FileManager.DownloadFile(w, s.Ctx, objectID, crc32)
	if err != nil {
		slog.Error("Ошибка при обработке файла", "object_id", objectID, "error", err)
		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
		return
	}
}

// handler := FileHandler(handleFile)
// if err := handler(w, fileName, rc, contentType); err != nil {
// 	slog.Error("Ошибка при обработке файла", "file", fileName, "error", err)
// 	http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 	return
// }

// s.FileManager.DownloadFile(r.Context(), objectID, crc32)
