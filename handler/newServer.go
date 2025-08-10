package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
)

type FileManager interface {
	UploadFile(r *http.Request, ctx context.Context, objectID, contentType, originalName string) error
	DownloadFile(w http.ResponseWriter, ctx context.Context, objectID string, crc uint32) error
	DeleteFile(objectID string) error
}

type Server struct {
	HTTPServer  *http.Server
	Ctx         context.Context
	FileManager FileManager
}

func NewServer(ctx context.Context, fm FileManager, port int) *Server {
	s := &Server{
		Ctx:         ctx,
		FileManager: fm,
	}
	r := s.setupRouter()

	s.HTTPServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	slog.Info("Сервер успешно создан", "address", s.HTTPServer.Addr)

	return s
}

func (s *Server) setupRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/{storage_name}/{relative_path}/objects/{object_id}/content", s.Upload)
	router.Get("/{storage_name}/{relative_path}/objects/{object_id}/content", s.Download)
	return router
}