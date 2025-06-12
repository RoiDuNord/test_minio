package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	objectID := strings.TrimPrefix(r.URL.Path, "/objects/download/")

	slog.Info("Начало обработки запроса на скачивание")

	minioClient := s.MinioClient.Client
	bucketName := s.MinioClient.BucketName

	stat, err := minioClient.StatObject(r.Context(), bucketName, objectID, minio.StatObjectOptions{})
	if err != nil {
		slog.Error("Ошибка при получении метаданных объекта", "error", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	originalName := stat.UserMetadata["X-Original-Name"]
	slog.Info("Имя", "name", originalName)
	if originalName == "" {
		if exts, _ := mime.ExtensionsByType(stat.ContentType); len(exts) > 0 {
			originalName = fmt.Sprintf("file_%d%s", time.Now().Unix(), exts[0])
		} else {
			originalName = objectID
		}
		slog.Info("Сгенерировано имя файла для скачивания", "fileName", originalName)
	} else {
		slog.Info("Найдено оригинальное имя файла в метаданных", "original_name", originalName)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, originalName))
	w.Header().Set("Content-Type", stat.ContentType)
	slog.Info("Установлены заголовки ответа")

	downloadTime := time.Now()
	object, err := minioClient.GetObject(r.Context(), bucketName, objectID, minio.GetObjectOptions{})
	if err != nil {
		slog.Error("Ошибка при получении объекта из MinIO", "error", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := object.Close(); err != nil {
			slog.Error("Ошибка при закрытии объекта", "error", err)
		}
	}()

	bytesWritten, err := io.Copy(w, object)
	if err != nil {
		slog.Error("Ошибка при отправке файла клиенту", "error", err)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("Файл успешно отправлен клиенту", "bytesWritten", bytesWritten, "duration", duration, "downloadTime", downloadDuration)
}

// package handlers

// import (
// 	"fmt"
// 	"io"
// 	"mime"
// 	"net/http"
// 	"strings"
// 	"time"

// 	"github.com/minio/minio-go/v7"
// 	"log/slog"
// )

// const (
// 	downloadBufferSize = 64 << 10
// 	downloadPathPrefix = "/objects/download/"
// )

// func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
// 	startTime := time.Now()
// 	objectID := strings.TrimPrefix(r.URL.Path, downloadPathPrefix)

// 	// Параллельно получаем метаданные
// 	statCh := make(chan *minio.ObjectInfo, 1)
// 	errCh := make(chan error, 1)
// 	go func() {
// 		stat, err := s.MinioClient.Client.StatObject(r.Context(), s.MinioClient.BucketName, objectID, minio.StatObjectOptions{})
// 		if err != nil {
// 			errCh <- err
// 			return
// 		}
// 		statCh <- &stat
// 	}()

// 	select {
// 	case stat := <-statCh:
// 		// Определяем имя файла
// 		fileName := stat.UserMetadata["X-Original-Name"]
// 		if fileName == "" {
// 			if exts, _ := mime.ExtensionsByType(stat.ContentType); len(exts) > 0 {
// 				fileName = fmt.Sprintf("file_%d%s", time.Now().Unix(), exts[0])
// 			} else {
// 				fileName = objectID
// 			}
// 		}

// 		// Устанавливаем заголовки
// 		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
// 		w.Header().Set("Content-Type", stat.ContentType)
// 		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))

// 		// Скачивание с буферизацией
// 		object, err := s.MinioClient.Client.GetObject(r.Context(), s.MinioClient.BucketName, objectID, minio.GetObjectOptions{})
// 		if err != nil {
// 			slog.Error("Failed to get object from MinIO", "objectID", objectID, "error", err)
// 			http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 			return
// 		}
// 		defer object.Close()

// 		// Используем буфер 64KB для экономии памяти
// 		buf := make([]byte, downloadBufferSize)
// 		if _, err := io.CopyBuffer(w, object, buf); err != nil {
// 			slog.Error("Failed to copy object to response", "objectID", objectID, "error", err)
// 			http.Error(w, "Failed to send file", http.StatusInternalServerError)
// 			return
// 		}

// 		slog.Info("File downloaded successfully", "objectID", objectID, "size_MB", stat.Size/(1<<20), "duration", time.Since(startTime))

// 	case err := <-errCh:
// 		slog.Error("Failed to get object metadata", "objectID", objectID, "error", err)
// 		http.Error(w, fmt.Sprintf("File not found: %v", err), http.StatusNotFound)
// 		return
// 	}
// }
