package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

const (
	maxUploadSize = 300 << 20
	statusCreated = "created"
	uploadMessage = "файл успешно загружен"
)

type ObjectResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	Message        string  `json:"message"`
	Duration       float64 `json:"duration_sec"`
	UploadDuration float64 `json:"uploadDuration_sec"`
}

func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	slog.Info("Начало обработки запроса на загрузку")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		slog.Error("Ошибка чтения тела запроса", "error", err)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	originalName := "default_name.bin"
	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				originalName = name
			}
		}
	}

	if originalName == "default_name.bin" {
		ext := filepath.Ext(originalName)
		if ext == "" {
			exts, _ := mime.ExtensionsByType(contentType)
			if len(exts) > 0 {
				ext = exts[0]
			}
		}
		originalName = fmt.Sprintf("file_%d%s", time.Now().Unix(), ext)
		slog.Info("Сгенерировано имя файла", "fileName", originalName)
	}

	objectID := uuid.New().String()

	uploadStart := time.Now()
	_, err = s.MinioClient.Client.PutObject(
		r.Context(),
		s.MinioClient.BucketName,
		objectID,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
				"X-Original-Name": originalName,
			},
		},
	)
	if err != nil {
		http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
		slog.Error("Ошибка загрузки в MinIO", "error", err)
		return
	}

	duration := time.Since(startTime)
	uploadDuration := time.Since(uploadStart)

	slog.Info("Файл успешно загружен")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	objectResponse := &ObjectResponse{
		ID:             objectID,
		Name:           originalName,
		Type:           contentType,
		Status:         statusCreated,
		Message:        uploadMessage,
		Duration:       duration.Seconds(),
		UploadDuration: uploadDuration.Seconds(),
	}
	if err := json.NewEncoder(w).Encode(objectResponse); err != nil {
		slog.Error("Ошибка формирования JSON ответа", "error", err)
	}
}

// 2 вариант
// package handlers

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"mime"
// 	"net/http"
// 	"path/filepath"
// 	"runtime"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/minio/minio-go/v7"
// )

// const (
// 	maxUploadSize = 400 << 20
// 	statusCreated = "created"
// 	uploadMessage = "Файл успешно загружен"
// 	uploadTimeout = 5 * time.Minute
// )

// type ObjectResponse struct {
// 	ID             string        `json:"id"`
// 	Name           string        `json:"name"`
// 	Type           string        `json:"type"`
// 	Status         string        `json:"status"`
// 	Message        string        `json:"message"`
// 	Duration       time.Duration `json:"duration_ms"`
// 	UploadDuration time.Duration `json:"uploadDuration_ms"`
// 	Size           int64         `json:"size_bytes"`
// }

// func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
// 	ctx, cancel := context.WithTimeout(r.Context(), uploadTimeout)
// 	defer cancel()

// 	startTime := time.Now()
// 	slog.Info("Начало обработки запроса на загрузку")

// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		slog.Error("Недопустимый метод запроса")
// 		return
// 	}

// 	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

// 	contentType := r.Header.Get("Content-Type")
// 	if contentType == "" {
// 		buf := make([]byte, 512)
// 		n, _ := io.ReadFull(r.Body, buf)
// 		contentType = http.DetectContentType(buf[:n])
// 		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf[:n]), r.Body))
// 	}

// 	originalName := "default_name.bin"
// 	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
// 		_, params, err := mime.ParseMediaType(contentDisposition)
// 		if err == nil {
// 			if name, ok := params["filename"]; ok && name != "" {
// 				originalName = name
// 			}
// 		}
// 	}

// 	if originalName == "default_name.bin" {
// 		ext := filepath.Ext(originalName)
// 		if ext == "" {
// 			exts, _ := mime.ExtensionsByType(contentType)
// 			if len(exts) > 0 {
// 				ext = exts[0]
// 			}
// 		}
// 		originalName = fmt.Sprintf("file_%d%s", time.Now().Unix(), ext)
// 		slog.Info("Сгенерировано имя файла", "fileName", originalName)
// 	}

// 	data, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, "Failed to read request body", http.StatusBadRequest)
// 		slog.Error("Ошибка чтения тела запроса", "error", err)
// 		return
// 	}

// 	objectID := uuid.New().String()
// 	size := int64(len(data))

// 	uploadStart := time.Now()
// 	_, err = s.MinioClient.Client.PutObject(
// 		ctx,
// 		s.MinioClient.BucketName,
// 		objectID,
// 		bytes.NewReader(data),
// 		size,
// 		minio.PutObjectOptions{
// 			ContentType: contentType,
// 			NumThreads:  uint(runtime.NumCPU()),
// 			UserMetadata: map[string]string{
// 				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
// 				"X-Original-Name": originalName,
// 				"X-File-Size":     fmt.Sprintf("%d", size),
// 			},
// 		},
// 	)
// 	if err != nil {
// 		http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
// 		slog.Error("Ошибка загрузки в MinIO", "error", err)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	uploadDuration := time.Since(uploadStart)

// 	slog.Info("Файл успешно загружен")

//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(http.StatusCreated)
//		objectResponse := &ObjectResponse{
//			ID:             objectID,
//			Name:           originalName,
//			Type:           contentType,
//			Status:         statusCreated,
//			Message:        uploadMessage,
//			Duration:       duration,
//			UploadDuration: uploadDuration,
//			Size:           size,
//		}
//		if err := json.NewEncoder(w).Encode(objectResponse); err != nil {
//			slog.Error("Ошибка формирования JSON ответа", "error", err)
//		}
// //	}

// 3 вариант
// package handlers

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"mime"
// 	"net/http"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/minio/minio-go/v7"
// )

// const (
// 	maxUploadSize  = 400 << 20 // 400 MB
// 	statusCreated  = "created"
// 	uploadMessage  = "Файл успешно загружен"
// 	uploadTimeout  = 5 * time.Minute
// 	defaultThreads = 16 // Значение по умолчанию для параллельной загрузки
// )

// type ObjectResponse struct {
// 	ID             string        `json:"id"`
// 	Name           string        `json:"name"`
// 	Type           string        `json:"type"`
// 	Status         string        `json:"status"`
// 	Message        string        `json:"message"`
// 	Duration       time.Duration `json:"duration_ms"`
// 	UploadDuration time.Duration `json:"uploadDuration_ms"`
// 	Size           int64         `json:"size_bytes"`
// }

// func (s *Server) Upload(w http.ResponseWriter, r *http.Request) {
// 	ctx, cancel := context.WithTimeout(r.Context(), uploadTimeout)
// 	defer cancel()

// 	startTime := time.Now()
// 	slog.Info("Начало обработки запроса на загрузку")

// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		slog.Error("Недопустимый метод запроса", "method", r.Method)
// 		return
// 	}

// 	// Проверяем Content-Length, если он указан, чтобы избежать лишней обработки
// 	contentLength := r.ContentLength
// 	if contentLength > maxUploadSize {
// 		errorMsg := fmt.Sprintf("Файл слишком большой. Максимальный допустимый размер: %d байт", maxUploadSize)
// 		http.Error(w, errorMsg, http.StatusRequestEntityTooLarge)
// 		slog.Error("Файл слишком большой", "contentLength", contentLength, "maxSize", maxUploadSize)
// 		return
// 	}

// 	// Ограничиваем размер тела запроса
// 	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

// 	// Определяем Content-Type из заголовка или через DetectContentType
// 	contentType := r.Header.Get("Content-Type")
// 	if contentType == "" {
// 		buf := make([]byte, 512)
// 		n, err := io.ReadAtLeast(r.Body, buf, 1)
// 		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
// 			http.Error(w, "Failed to read request body for content detection", http.StatusBadRequest)
// 			slog.Error("Ошибка чтения тела запроса для определения типа", "error", err)
// 			return
// 		}
// 		contentType = http.DetectContentType(buf[:n])
// 		// Возвращаем прочитанные данные в поток
// 		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf[:n]), r.Body))
// 	}

// 	// Определяем оригинальное имя файла
// 	originalName := "default_name.bin"
// 	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
// 		_, params, err := mime.ParseMediaType(contentDisposition)
// 		if err == nil {
// 			if name, ok := params["filename"]; ok && name != "" {
// 				originalName = name
// 			}
// 		}
// 	}

// 	if originalName == "default_name.bin" {
// 		ext := filepath.Ext(originalName)
// 		if ext == "" {
// 			if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
// 				ext = exts[0]
// 			}
// 		}
// 		originalName = fmt.Sprintf("file_%d%s", time.Now().Unix(), ext)
// 		slog.Info("Сгенерировано имя файла", "fileName", originalName)
// 	}

// 	// Генерируем уникальный ID для объекта
// 	objectID := uuid.New().String()

// 	// Загружаем файл напрямую в MinIO без чтения в память
// 	uploadStart := time.Now()
// 	info, err := s.MinioClient.Client.PutObject(
// 		ctx,
// 		s.MinioClient.BucketName,
// 		objectID,
// 		r.Body,
// 		-1, // Размер неизвестен заранее, MinIO сам определит
// 		minio.PutObjectOptions{
// 			ContentType: contentType,
// 			NumThreads:  defaultThreads, // Используем фиксированное значение или настраиваемое
// 			UserMetadata: map[string]string{
// 				"X-Uploaded-At":   time.Now().Format(time.RFC3339),
// 				"X-Original-Name": originalName,
// 			},
// 		},
// 	)
// 	if err != nil {
// 		// Проверяем, связана ли ошибка с ограничением размера (Nginx 413)
// 		errorMsg := err.Error()
// 		if strings.Contains(errorMsg, "413 Request Entity Too Large") {
// 			http.Error(w, "Файл слишком большой для сервера. Пожалуйста, уменьшите размер файла или обратитесь к администратору.", http.StatusRequestEntityTooLarge)
// 			slog.Error("Ошибка загрузки в MinIO: ограничение размера на сервере (Nginx 413)", "objectID", objectID, "error", errorMsg)
// 		} else {
// 			http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
// 			slog.Error("Ошибка загрузки в MinIO", "objectID", objectID, "error", errorMsg)
// 		}
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	uploadDuration := time.Since(uploadStart)

// 	slog.Info("Файл успешно загружен", "objectID", objectID, "size_bytes", info.Size)

// 	// Формируем и отправляем ответ
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	objectResponse := &ObjectResponse{
// 		ID:             objectID,
// 		Name:           originalName,
// 		Type:           contentType,
// 		Status:         statusCreated,
// 		Message:        uploadMessage,
// 		Duration:       duration,
// 		UploadDuration: uploadDuration,
// 		Size:           info.Size,
// 	}
// 	if err := json.NewEncoder(w).Encode(objectResponse); err != nil {
// 		slog.Error("Ошибка формирования JSON ответа", "error", err)
// 	}
// }
