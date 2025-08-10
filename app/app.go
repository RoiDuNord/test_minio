package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"test_minio/config"
	"test_minio/handler"
	"test_minio/minio"
)

func Run() error {
	ctx, cancel := initContext()
	defer cancel()

	cfg, err := config.Get()
	if err != nil {
		return err
	}

	fmt.Println(cfg)

	minio, err := minio.Init(cfg.MinIO)
	if err != nil {
		return err
	}

	err = minio.CreateBucket(ctx, cfg.MinIO.Location)
	if err != nil {
		return err
	}

	server := handler.NewServer(ctx, minio, cfg.App.Port)

	go startHTTPServer(server.HTTPServer, server.HTTPServer.Addr)

	return shutdownServer(server)

}

func initContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func startHTTPServer(server *http.Server, port string) {
	slog.Info(fmt.Sprintf("starting HTTP server on port %s", port[1:]))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("error starting HTTP server", "error", err)
	}
}

func shutdownServer(s *handler.Server) error {
	server := s.HTTPServer
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	select {
	case <-shutdownSignals:
		slog.Info("получен сигнал завершения работы")
	case <-s.Ctx.Done():
		slog.Info("истекло время ожидания контекста")
	}

	if err := server.Shutdown(s.Ctx); err != nil {
		slog.Error("не удалось корректно завершить работу сервера", "ошибка", err)

		if err := server.Close(); err != nil {
			slog.Error("не удалось принудительно завершить работу сервера", "ошибка", err)
			return err
		}
	}

	slog.Info("сервер успешно завершил работу")
	return nil
}
