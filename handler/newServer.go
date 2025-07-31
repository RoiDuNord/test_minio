package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
)

// type FileManager interface {
// 	UploadFile(objectID string, data io.Reader) error
// 	DownloadFile(objectID string) error
// 	DeleteFile(objectID string) error
// }

type FileManager interface {
	UploadFile(objectID string, data io.Reader) error
	DownloadFile(w http.ResponseWriter, ctx context.Context, objectID string, crc uint32) error
	DeleteFile(objectID string) error
}

type Server struct {
	HTTPServer  *http.Server
	Ctx         context.Context
	FileManager FileManager
	Router      *chi.Mux
}

func NewServer(ctx context.Context, fm FileManager, bucketName string, port int) *Server {
	router := chi.NewRouter()

	s := &Server{
		Router:      router,
		Ctx:         ctx,
		FileManager: fm,
	}

	setupRoutes(s)

	s.HTTPServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	slog.Info("Сервер успешно создан", "address", s.HTTPServer.Addr)

	return s
}

func setupRoutes(s *Server) {
	s.Router.Post("{storage_name}/{relative_path}/objects/{object_id}/content", s.Upload)
	s.Router.Get("{storage_name}/{relative_path}/objects/{object_id}/content", s.Download)
}
