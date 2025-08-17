package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
)

type DownloadRequestMetadata struct {
	ID    string
	CRC32 uint32
}

func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
	slog.Info("Начало обработки запроса на скачивание")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	parsedData := chi.URLParam(r, "object_id")
	downloadData, err := getIDandCRC(parsedData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Не удалось извлечь object_id и crc32", "error", err)
		return
	}

	if err := s.loadManager.Download(w, s.ctx, downloadData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
