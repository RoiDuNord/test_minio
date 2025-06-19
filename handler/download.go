package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/minio/minio-go/v7"
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
	startTime := time.Now()
	objectID := chi.URLParam(r, "object_id")
	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID)

	stat, err := s.getObjectStat(r.Context(), objectID)
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	if stat.ContentType == "application/zip" {
		s.handleZipFile(w, r, objectID, stat, startTime)
		return
	}
	s.handleRegularFile(w, r, objectID, stat, startTime)
}

type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error

func getContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	defaultContentType := "application/octet-stream"
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return defaultContentType
}

func handleFile(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error {
	slog.Info("Отправка файла клиенту", "file", fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	_, err := io.Copy(w, content)
	if err != nil {
		return fmt.Errorf("не удалось отправить данные файла: %v", err)
	}
	return nil
}

func readObjectToBuffer(object io.Reader) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, object)
	return buf, err
}

func findSuitableFile(zipReader *zip.Reader) (*zip.File, error) {
	for _, file := range zipReader.File {
		if !file.FileInfo().IsDir() && !strings.HasPrefix(filepath.Base(file.Name), ".") {
			return file, nil
		}
	}
	return nil, fmt.Errorf("в ZIP-архиве не найдено подходящего файла")
}

// processZip обрабатывает ZIP-архив
func processZip(w http.ResponseWriter, r *http.Request, object io.Reader, size int64) error {
	slog.Info("Обработка ZIP-архива")
	buf, err := readObjectToBuffer(object)
	if err != nil {
		return fmt.Errorf("не удалось прочитать данные ZIP: %v", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), size)
	if err != nil {
		return fmt.Errorf("не удалось прочитать ZIP-архив: %v", err)
	}

	innerFile, err := findSuitableFile(zipReader)
	if err != nil {
		return err
	}

	rc, err := innerFile.Open()
	if err != nil {
		return fmt.Errorf("не удалось открыть файл %s в ZIP: %v", innerFile.Name, err)
	}
	defer rc.Close()

	contentType := getContentType(innerFile.Name)
	handler := FileHandler(handleFile)
	return handler(w, innerFile.Name, rc, contentType)
}

func (s *Server) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
	return s.MinioClient.Client.StatObject(ctx, s.MinioClient.BucketName, objectID, minio.StatObjectOptions{})
}

func (s *Server) getObject(ctx context.Context, objectID string) (*minio.Object, error) {
	return s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectID, minio.GetObjectOptions{})
}

func (s *Server) determineFileName(stat minio.ObjectInfo) string {
	originalName := stat.UserMetadata["X-Original-Name"]
	if originalName != "" {
		slog.Info("Найдено оригинальное имя файла в метаданных", "original_name", originalName)
		return originalName
	}
	return stat.Key
}

func (s *Server) handleRegularFile(w http.ResponseWriter, r *http.Request, objectID string, stat minio.ObjectInfo, startTime time.Time) {
	fileName := s.determineFileName(stat)
	downloadTime := time.Now()

	object, err := s.getObject(r.Context(), objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		http.Error(w, "Не удалось скачать файл", http.StatusInternalServerError)
		return
	}
	defer object.Close()

	handler := FileHandler(handleFile)
	if err := handler(w, fileName, object, stat.ContentType); err != nil {
		slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
		http.Error(w, "Не удалось отправить данные файла", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("Файл отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
}

// handleZipFile обрабатывает ZIP-архив
func (s *Server) handleZipFile(w http.ResponseWriter, r *http.Request, objectID string, stat minio.ObjectInfo, startTime time.Time) {
	downloadTime := time.Now()
	object, err := s.getObject(r.Context(), objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		http.Error(w, "Не удалось скачать файл", http.StatusInternalServerError)
		return
	}
	defer object.Close()

	err = processZip(w, r, object, stat.Size)
	if err != nil {
		slog.Error("Не удалось обработать ZIP-архив", "object_id", objectID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
}
