package server

import (
	"log/slog"
	"net/http"
)

type objectResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int    `json:"size_mb"`
}

type UploadRequestMetadata struct {
	ID          string
	FileName    string
	ContentType string
	Size        int64
}

func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	slog.Info("Начало обработки запроса на загрузку")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	data, err := getUploadRequestData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if err := s.loadManager.Upload(r, s.ctx, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, data)
}
