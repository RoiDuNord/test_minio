package handler

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/minio/minio-go/v7"
)

type MinioClient struct {
	Client     *minio.Client
	BucketName string
}

type Server struct {
	HTTPServer  *http.Server
	Ctx         context.Context
	MinioClient MinioClient
	Router      *chi.Mux
	Objects     sync.Map
}

func NewServer(ctx context.Context, minioClient *minio.Client, bucketName string) *Server {
	router := chi.NewRouter()

	client := &MinioClient{
		Client:     minioClient,
		BucketName: bucketName,
	}

	s := &Server{
		Router:      router,
		Ctx:         ctx,
		MinioClient: *client,
	}

	setupRoutes(s)

	s.HTTPServer = &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	slog.Info("Сервер успешно создан", "address", s.HTTPServer.Addr)

	return s
}

func setupRoutes(s *Server) {
	s.Router.Post("/objects/{object_id}/content", s.Upload)
	s.Router.Get("/objects/{object_id}/content", s.Download)
}
