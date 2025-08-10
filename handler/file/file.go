package file

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"test_minio/models"
	"time"

	"github.com/minio/minio-go/v7"
)

const DefaultUploadFileName = "default_name.bin"

func ParseObjectIDandCRC(parsedInfo string) (objectID string, crc uint32, err error) {
	parsedInfo = strings.TrimSpace(parsedInfo)
	if parsedInfo == "" {
		err = fmt.Errorf("object identifier is required")
		slog.Error("Пустой идентификатор объекта", "error", err)
		return "", 0, err
	}

	parts := strings.Split(parsedInfo, ";")
	if len(parts) > 2 {
		err = fmt.Errorf("invalid format: expected 'objectID;crc32'")
		slog.Error("Неверный формат идентификатора", "error", err, "parsed_info", parsedInfo)
		return "", 0, err
	}

	objectID = strings.TrimSpace(parts[0])
	if objectID == "" {
		err = fmt.Errorf("object identifier is required")
		slog.Error("Пустой идентификатор объекта после разделения", "error", err, "parsed_info", parsedInfo)
		return "", 0, err
	}

	if len(parts) == 2 {
		crcStr := strings.TrimSpace(parts[1])
		crcValue, parseErr := strconv.ParseUint(crcStr, 10, 32)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse CRC value: %v", parseErr)
			slog.Error("Ошибка разбора значения CRC", "error", err, "crc_info", crcStr)
			return "", 0, err
		}
		crc = uint32(crcValue)
	}

	return objectID, crc, nil
}

func findSuitableFile(zipReader *zip.Reader, crc32 uint32) (*zip.File, error) {
	for _, file := range zipReader.File {
		fmt.Println("crc32 of file", file.CRC32)
		if file.CRC32 == crc32 {
			return file, nil
		}
	}
	return nil, fmt.Errorf("file with CRC32 %d not found in ZIP archive", crc32)
}

func processZip(w http.ResponseWriter, object io.ReaderAt, size int64, crc uint32) error {
	slog.Info("Начало обработки ZIP-архива", "размер", size)
	zipReader, err := zip.NewReader(object, size)
	if err != nil {
		slog.Error("Ошибка при чтении ZIP-архива", "error", err)
		return fmt.Errorf("failed to read ZIP archive: %v", err)
	}
	slog.Info("Успешно создан читатель ZIP-архива", "количество_файлов", len(zipReader.File))

	searchedFile, err := findSuitableFile(zipReader, crc)
	if err != nil {
		slog.Error("Не удалось найти подходящий файл в ZIP", "error", err, "crc", crc)
		return err
	}
	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc)

	rc, err := searchedFile.Open()
	if err != nil {
		slog.Error("Ошибка при открытии файла в ZIP", "file", searchedFile.Name, "error", err)
		return fmt.Errorf("failed to open file %s in ZIP: %v", searchedFile.Name, err)
	}
	defer rc.Close()
	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

	contentType := GetContentType(searchedFile.Name)
	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "тип_содержимого", contentType)

	handler := FileHandler(handleFile)
	if err := handler(w, searchedFile.Name, rc, contentType); err != nil {
		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
		return err
	}

	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
	return nil
}

func GetContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	defaultContentType := "application/octet-stream"
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return defaultContentType
}

func determineFileName(stat minio.ObjectInfo) string {
	originalName := stat.UserMetadata["X-Original-Name"]
	if originalName != "" {
		return originalName
	}
	return stat.Key
}

type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error // есть ли в этом смысл?

func handleFile(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error {
	slog.Info("Начало установки заголовков", "fileName", fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	slog.Info("Заголовки установлены")

	startTime := time.Now()
	slog.Info("Начало передачи данных клиенту", "fileName", fileName)
	pw := &models.ProgressWriter{Writer: w, LastLogTime: time.Now()}
	_, err := io.Copy(pw, content)
	if err != nil {
		slog.Error("Ошибка при передаче данных", "fileName", fileName, "error", err)
		return fmt.Errorf("failed to send file data: %v", err)
	}
	slog.Info("Данные переданы клиенту", "fileName", fileName, "duration", time.Since(startTime).Seconds(), "total_bytes", pw.Total)
	return nil
}

func HandleRegularFile(w http.ResponseWriter, ctx context.Context, objectID string, object *minio.Object, stat minio.ObjectInfo, startTime time.Time) {
	defer object.Close()

	fileName := determineFileName(stat)
	downloadTime := time.Now()

	handler := FileHandler(handleFile)
	if err := handler(w, fileName, object, stat.ContentType); err != nil {
		slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("Файл отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
}

func HandleZipFile(w http.ResponseWriter, ctx context.Context, objectID string, object *minio.Object, stat minio.ObjectInfo, crc32 uint32, size int64, startTime time.Time) {
	defer object.Close()

	downloadTime := time.Now()

	err := processZip(w, object, size, crc32)
	if err != nil {
		slog.Error("Ошибка обработки ZIP-архива", "object_id", objectID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	downloadDuration := time.Since(downloadTime)
	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
}

// func GenerateFileName(contentType string) string {
// 	originalName := defaultFileName
// 	ext := filepath.Ext(originalName)
// 	if ext == "" {
// 		exts, err := mime.ExtensionsByType(contentType)
// 		if err != nil {
// 			slog.Warn("Не удалось определить расширение по Content-Type", "content_type", contentType, "error", err)
// 			ext = ".bin"
// 		} else if len(exts) > 0 {
// 			ext = exts[0]
// 		} else {
// 			ext = ".bin"
// 		}
// 		originalName = uuid.NewString()
// 	}
// 	return originalName
// }
