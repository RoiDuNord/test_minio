// package handler

// import (
// 	"archive/zip"
// 	"bytes"
// 	"strconv"

// 	"context"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"mime"
// 	"net/http"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"github.com/go-chi/chi"
// 	"github.com/minio/minio-go/v7"
// )

// // Download обрабатывает запрос на скачивание файла из MinIO
// // @Summary Download content for an object
// // @Description Downloads the content of the specified object ID from MinIO. Supports both regular files and ZIP archives.
// // @Tags Objects
// // @Produce application/octet-stream
// // @Param object_id path string true "Object ID"
// // @Success 200 "File downloaded successfully"
// // @Failure 400 {object} string "Invalid request"
// // @Failure 404 {object} string "Object not found"
// // @Failure 500 {object} string "Internal server error"
// // @Router /objects/{object_id}/content [get]
// func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
// 	if err := checkGetMethod(r); err != nil {
// 		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
// 		slog.Error("Недопустимый метод запроса", "error", err)
// 		return
// 	}

// 	startTime := time.Now()
// 	objectID, crc, err := parseObjectIDandCRC(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID)

// 	stat, err := s.getObjectStat(r.Context(), objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
// 		http.Error(w, "File not found", http.StatusNotFound)
// 		return
// 	}

// 	if stat.ContentType == "application/zip" {
// 		if crc == 0 {
// 			slog.Error("Отсутствует CRC для ZIP", "object_id", objectID)
// 			http.Error(w, "No CRC for ZIP", http.StatusBadRequest)
// 			return
// 		}
// 		s.handleZipFile(w, r, objectID, crc, stat.Size, startTime)
// 		return
// 	}
// 	s.handleRegularFile(w, r, objectID, stat, startTime)
// }

// type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error

// func checkGetMethod(r *http.Request) error {
// 	if r.Method != http.MethodGet {
// 		return fmt.Errorf("method not allowed: %s", r.Method)
// 	}
// 	return nil
// }

// func parseObjectIDandCRC(r *http.Request) (objectID string, crc uint32, err error) {
// 	objectID = chi.URLParam(r, "object_id")
// 	if objectID == "" {
// 		err = fmt.Errorf("object identifier is required")
// 		slog.Error("Пустой идентификатор объекта", "error", err)
// 		return "", 0, err
// 	}

// 	if strings.Contains(objectID, ";") {
// 		parts := strings.Split(objectID, ";")
// 		if len(parts) != 2 {
// 			err = fmt.Errorf("invalid format: expected 'fileID;crc' for ZIP archive")
// 			slog.Error("Неверный формат идентификатора для ZIP", "error", err, "input", objectID)
// 			return "", 0, err
// 		}

// 		objectID = parts[0]
// 		if objectID == "" {
// 			err = fmt.Errorf("object identifier is required")
// 			slog.Error("Пустой идентификатор объекта в формате ZIP", "error", err, "input", objectID)
// 			return "", 0, err
// 		}

// 		crcValue, err := strconv.ParseUint(parts[1], 10, 32)
// 		if err != nil {
// 			err = fmt.Errorf("failed to parse CRC value: %v", err)
// 			slog.Error("Ошибка разбора значения CRC", "error", err, "input", parts[1])
// 			return "", 0, err
// 		}

// 		return objectID, uint32(crcValue), nil
// 	}

// 	return objectID, 0, nil
// }

// func getContentType(fileName string) string {
// 	ext := strings.ToLower(filepath.Ext(fileName))
// 	defaultContentType := "application/octet-stream"
// 	if contentType := mime.TypeByExtension(ext); contentType != "" {
// 		return contentType
// 	}
// 	return defaultContentType
// }

// type ProgressWriter struct {
// 	w     io.Writer
// 	total int64
// 	last  time.Time
// }

// func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
// 	n, err = pw.w.Write(p)
// 	pw.total += int64(n)
// 	if time.Since(pw.last) > 1*time.Second {
// 		slog.Info("Прогресс передачи данных", "total_bytes", pw.total)
// 		pw.last = time.Now()
// 	}
// 	return n, err
// }

// func handleFile(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error {
// 	start := time.Now()
// 	slog.Info("Начало установки заголовков", "fileName", fileName)
// 	w.Header().Set("Content-Type", contentType)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
// 	slog.Info("Заголовки установлены", "duration", time.Since(start).Seconds())

// 	start = time.Now()
// 	slog.Info("Начало передачи данных клиенту", "fileName", fileName)
// 	pw := &ProgressWriter{w: w, last: time.Now()}
// 	_, err := io.Copy(pw, content)
// 	if err != nil {
// 		slog.Error("Ошибка при передаче данных", "fileName", fileName, "error", err)
// 		return fmt.Errorf("failed to send file data: %v", err)
// 	}
// 	slog.Info("Данные переданы клиенту", "fileName", fileName, "duration", time.Since(start).Seconds(), "total_bytes", pw.total)
// 	return nil
// }

// func findSuitableFile(zipReader *zip.Reader, crc32 uint32) (*zip.File, error) {
// 	for _, file := range zipReader.File {
// 		fmt.Println("crc32 of file", file.CRC32)
// 		if file.CRC32 == crc32 {
// 			return file, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("file with CRC32 %d not found in ZIP archive", crc32)
// }

// func processZip(w http.ResponseWriter, r *http.Request, object io.ReaderAt, size int64, crc uint32) error {
// 	slog.Info("Начало обработки ZIP-архива", "размер", size)

// 	// Создаем буфер в памяти для хранения данных из object
// 	buf := bytes.NewBuffer(make([]byte, 0, size))
// 	tempReader := io.NewSectionReader(object, 0, size)
// 	start := time.Now()
// 	_, err := io.Copy(buf, tempReader)
// 	if err != nil {
// 		slog.Error("Ошибка при копировании данных в буфер", "error", err)
// 		return fmt.Errorf("failed to copy data to buffer: %v", err)
// 	}
// 	slog.Info("Данные успешно скопированы в буфер", "время_копирования", time.Since(start).Seconds(), "размер_буфера", buf.Len())

// 	// Создаем новый io.ReaderAt из буфера для zip.NewReader
// 	bufferedReaderAt := bytes.NewReader(buf.Bytes())

// 	// Создаем читатель ZIP-архива из буферизованных данных
// 	zipReader, err := zip.NewReader(bufferedReaderAt, size)
// 	if err != nil {
// 		slog.Error("Ошибка при чтении ZIP-архива", "error", err)
// 		return fmt.Errorf("failed to read ZIP archive: %v", err)
// 	}
// 	slog.Info("Успешно создан читатель ZIP-архива", "количество_файлов", len(zipReader.File))

// 	searchedFile, err := findSuitableFile(zipReader, crc)
// 	if err != nil {
// 		slog.Error("Не удалось найти подходящий файл в ZIP", "error", err, "crc", crc)
// 		return err
// 	}
// 	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc)

// 	rc, err := searchedFile.Open()
// 	if err != nil {
// 		slog.Error("Ошибка при открытии файла в ZIP", "file", searchedFile.Name, "error", err)
// 		return fmt.Errorf("failed to open file %s in ZIP: %v", searchedFile.Name, err)
// 	}
// 	defer rc.Close()
// 	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

// 	contentType := getContentType(searchedFile.Name)
// 	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "тип_содержимого", contentType)

// 	handler := FileHandler(handleFile)
// 	if err := handler(w, searchedFile.Name, rc, contentType); err != nil {
// 		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return err
// 	}

// 	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
// 	return nil
// }

// func (s *Server) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
// 	return s.MinioClient.Client.StatObject(ctx, s.MinioClient.BucketName, objectID, minio.StatObjectOptions{})
// }

// func (s *Server) getObject(ctx context.Context, objectID string) (*minio.Object, error) {
// 	return s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectID, minio.GetObjectOptions{})
// }

// func (s *Server) determineFileName(stat minio.ObjectInfo) string {
// 	originalName := stat.UserMetadata["X-Original-Name"]
// 	if originalName != "" {
// 		return originalName
// 	}
// 	return stat.Key
// }

// func (s *Server) handleRegularFile(w http.ResponseWriter, r *http.Request, objectID string, stat minio.ObjectInfo, startTime time.Time) {
// 	fileName := s.determineFileName(stat)
// 	downloadTime := time.Now()

// 	object, err := s.getObject(s.Ctx, objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 		return
// 	}
// 	defer object.Close()

// 	handler := FileHandler(handleFile)
// 	if err := handler(w, fileName, object, stat.ContentType); err != nil {
// 		slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("Файл отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
// }

// func (s *Server) handleZipFile(w http.ResponseWriter, r *http.Request, objectID string, crc uint32, size int64, startTime time.Time) {
// 	downloadTime := time.Now()
// 	object, err := s.getObject(s.Ctx, objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 		return
// 	}
// 	defer object.Close()

// 	err = processZip(w, r, object, size, crc)
// 	if err != nil {
// 		slog.Error("Ошибка обработки ZIP-архива", "object_id", objectID, "error", err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
// }

package handler

// через буфер
// func processZip(w http.ResponseWriter, r *http.Request, object *minio.Object, size int64, crc32 uint32) error {
// 	slog.Info("Начало обработки ZIP-архива", "размер", size)

// 	buf := new(bytes.Buffer)
// 	_, err := io.Copy(buf, object)
// 	if err != nil {
// 		return fmt.Errorf("не удалось прочитать данные ZIP: %v", err)
// 	}

// 	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), size)
// 	if err != nil {
// 		return fmt.Errorf("не удалось прочитать ZIP-архив: %v", err)
// 	}

// 	searchedFile, err := findSuitableFile(zipReader, crc32)
// 	if err != nil {
// 		slog.Error("Не удалось найти подходящий файл в ZIP", "error", err, "crc32", crc32)
// 		return err
// 	}
// 	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc32)

// 	rc, err := searchedFile.Open()
// 	if err != nil {
// 		slog.Error("Ошибка при открытии файла в ZIP", "file", searchedFile.Name, "error", err)
// 		return fmt.Errorf("failed to open file %s in ZIP: %v", searchedFile.Name, err)
// 	}
// 	defer rc.Close()
// 	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

// 	contentType := getContentType(searchedFile.Name)
// 	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "contentType", contentType)

// 	handler := FileHandler(handleFile)
// 	if err := handler(w, searchedFile.Name, rc, contentType); err != nil {
// 		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return err
// 	}

// 	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
// 	return nil
// }
