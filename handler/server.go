package handler

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"s3_multiclient/config"
	"syscall"

	"github.com/go-chi/chi"
)

type LoadManager interface {
	Upload(r *http.Request, ctx context.Context, objectID, contentType, originalFileName string, contentLength int64) error
	Download(w http.ResponseWriter, ctx context.Context, objectID string, crc32 uint32) error
	Delete(w http.ResponseWriter, r *http.Request, ctx context.Context) error
}

// type DBManager interface{
// 	Upload(ctx context.Context, w http.ResponseWriter, r *http.Request)
// 	Download(ctx context.Context, w http.ResponseWriter, r *http.Request)
// 	Delete(ctx context.Context, w http.ResponseWriter, r *http.Request)
// }

// type DefaultLoadManager struct {
// 	fileManager load.FileManager
// 	dbManager   db.DBManager
// }

// func NewDefaultLoadManager(fm load.FileManager, dm db.DBManager) *DefaultLoadManager {
// 	return &DefaultLoadManager{
// 		fileManager: fm,
// 		dbManager:   dm,
// 	}
// }

type Server struct {
	ctx         context.Context
	loadManager LoadManager
	// dbManager DBManager
}

func NewServer(ctx context.Context, lm LoadManager) *Server {
	return &Server{
		ctx:         ctx,
		loadManager: lm,
	}
}

func (s *Server) setupRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/{storage_name}/{relative_path}/objects/{object_id}/content", s.Upload)
	router.Get("/{storage_name}/{relative_path}/objects/{object_id}/content", s.Download)
	return router
}

func (s *Server) Start(cfg config.AppConfig) error {
	router := s.setupRouter()
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	slog.Info(fmt.Sprintf("starting HTTP server on address %s", addr))
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error starting HTTP server", "error", err)
			log.Fatal("сервер не стартовал")
		}
	}()

	return s.gracefulShutdown(httpServer)
}

func (s *Server) gracefulShutdown(server *http.Server) error {
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	select {
	case <-shutdownSignals:
		slog.Info("получен сигнал завершения работы")
	case <-s.ctx.Done():
		slog.Info("истекло время ожидания контекста")
	}

	if err := server.Shutdown(s.ctx); err != nil {
		slog.Error("не удалось корректно завершить работу сервера", "ошибка", err)
		return err
	}

	slog.Info("сервер успешно завершил работу")
	return nil
}
