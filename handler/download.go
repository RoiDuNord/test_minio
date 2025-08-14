package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
)

func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
	slog.Info("Начало обработки запроса на скачивание")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	parsedInfo := chi.URLParam(r, "object_id")
	objectID, crc32, err := parseObjectIDandCRC(parsedInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Не удалось извлечь object_id и crc", "error", err)
		return
	}

	if err := s.loadManager.Download(w, s.ctx, objectID, crc32); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
