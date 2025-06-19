package handler

import (
	"archive/zip"
	"strconv"

	//	"bytes"
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
	if err := checkGetMethod(r); err != nil {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		slog.Error("Недопустимый метод запроса", "error", err)
		return
	}

	startTime := time.Now()
	objectID, crc, isZip, err := ParseObjectIDandCRC(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID)

	stat, err := s.getObjectStat(r.Context(), objectID)
	if err != nil {
		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	if isZip {
		s.handleZipFile(w, r, objectID, crc, stat.Size, startTime)
		return
	}
	s.handleRegularFile(w, r, objectID, stat, startTime)
}

type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error

func checkGetMethod(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
	return nil
}

func ParseObjectIDandCRC(r *http.Request) (objectID string, crc uint32, isZip bool, err error) {
	objectID = chi.URLParam(r, "object_id")
	if objectID == "" {
		err = fmt.Errorf("object identifier is required")
		slog.Error("Ошибка при разборе идентификатора файла: пустой идентификатор объекта", "error", err)
		return "", 0, false, err
	}

	if strings.Contains(objectID, ";") {
		parts := strings.Split(objectID, ";")
		if len(parts) != 2 {
			err = fmt.Errorf("invalid format: expected 'fileID;crc' for ZIP archive")
			slog.Error("Ошибка при разборе идентификатора файла для ZIP-архива", "error", err, "input", objectID)
			return "", 0, false, err
		}

		objectID = parts[0]
		if objectID == "" {
			err = fmt.Errorf("object identifier is required")
			slog.Error("Ошибка при разборе идентификатора файла: пустой идентификатор объекта в формате ZIP", "error", err, "input", objectID)
			return "", 0, false, err
		}

		crcValue, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			err = fmt.Errorf("failed to parse CRC value: %v", err)
			slog.Error("Ошибка при разборе значения CRC для ZIP-архива", "error", err, "input", parts[1])
			return "", 0, false, err
		}

		slog.Info("Успешно разобран идентификатор ZIP-архива", "object_id", objectID, "crc", crcValue)
		return objectID, uint32(crcValue), true, nil
	}

	slog.Info("Успешно разобран идентификатор обычного файла", "object_id", objectID)
	return objectID, 0, false, nil
}

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

func findSuitableFile(zipReader *zip.Reader, crc uint32) (*zip.File, error) {
	for _, file := range zipReader.File {
		notDir := isNotDirectory(file)
		notHidden := isNotHidden(file.Name)
		crcMatch := matchesCRC(file, crc)

		if notDir && notHidden && crcMatch {
			slog.Info("Найден подходящий файл", "name", file.Name, "crc", file.CRC32)
			return file, nil
		}
		slog.Debug("Файл отклонён", "name", file.Name, "isDir", !notDir, "hidden", !notHidden, "crcMatch", crcMatch)
	}
	slog.Error("В ZIP-архиве не найдено подходящего файла", "crc", crc)
	return nil, fmt.Errorf("no suitable file found in ZIP archive")
}

func isNotDirectory(file *zip.File) bool {
	return !file.FileInfo().IsDir()
}

// isNotHidden проверяет, что имя файла не начинается с точки (не скрытый файл).
func isNotHidden(fileName string) bool {
	baseName := filepath.Base(fileName)
	return !strings.HasPrefix(baseName, ".")
}

// matchesCRC проверяет, совпадает ли CRC32 файла с заданным значением.
func matchesCRC(file *zip.File, crc uint32) bool {
	return file.CRC32 == crc
}

// processZip обрабатывает ZIP-архив
func processZip(w http.ResponseWriter, r *http.Request, object io.ReaderAt, size int64, crc uint32) error {
	slog.Info("Начало обработки ZIP-архива", "размер", size)

	zipReader, err := zip.NewReader(object, size)
	if err != nil {
		slog.Error("Ошибка при чтении ZIP-архива", "ошибка", err)
		return fmt.Errorf("не удалось прочитать ZIP-архив: %v", err)
	}
	slog.Info("Успешно создан читатель ZIP-архива", "количество_файлов", len(zipReader.File))

	innerFile, err := findSuitableFile(zipReader, crc)
	if err != nil {
		slog.Error("Ошибка при поиске подходящего файла в ZIP-архиве", "ошибка", err, "crc32", crc)
		return err
	}
	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", innerFile.Name, "crc32", crc)

	rc, err := innerFile.Open()
	if err != nil {
		slog.Error("Ошибка при открытии файла в ZIP-архиве", "имя_файла", innerFile.Name, "ошибка", err)
		return fmt.Errorf("не удалось открыть файл %s в ZIP: %v", innerFile.Name, err)
	}
	defer rc.Close()
	slog.Info("Файл успешно открыт для чтения", "имя_файла", innerFile.Name)

	contentType := getContentType(innerFile.Name)
	slog.Info("Определен тип содержимого файла", "имя_файла", innerFile.Name, "тип_содержимого", contentType)

	handler := FileHandler(handleFile)
	err = handler(w, innerFile.Name, rc, contentType)
	if err != nil {
		slog.Error("Ошибка при обработке файла через обработчик", "имя_файла", innerFile.Name, "ошибка", err)
		return err
	}

	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", innerFile.Name)
	return nil
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
func (s *Server) handleZipFile(w http.ResponseWriter, r *http.Request, objectID string, crc uint32, size int64, startTime time.Time) {
	downloadTime := time.Now()
	object, err := s.getObject(r.Context(), objectID)
	if err != nil {
		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
		http.Error(w, "Не удалось скачать файл", http.StatusInternalServerError)
		return
	}
	defer object.Close()

	err = processZip(w, r, object, size, crc)
	if err != nil {
		// slog.Error("Не удалось обработать ZIP-архив", "object_id", objectID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
}
