package minio

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path/filepath"
	"s3_multiclient/load"
	"strings"

	"github.com/minio/minio-go/v7"
)

func handleRegularFile(pw *load.ProgressWriter, objectID string, object *minio.Object, stat minio.ObjectInfo) error {
	defer object.Close()

	fileName := determineFileName(stat)

	fileManager := FileHandler(handleFile)
	if err := fileManager(pw, fileName, object, stat.ContentType); err != nil {
		return fmt.Errorf("не удалось отправить файл клиенту: %w", err)
	}

	slog.Info("Файл отправлен клиенту", "object_id", objectID)
	return nil
}

func handleZipFile(pw *load.ProgressWriter, objectID string, object *minio.Object, crc32 uint32, size int64) error {
	defer object.Close()

	if err := processZip(pw, object, size, crc32); err != nil {
		return fmt.Errorf("ошибка обработки ZIP-архива: %w", err)
	}

	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID)
	return nil
}

type FileHandler func(pw *load.ProgressWriter, fileName string, content io.Reader, contentType string) error

func handleFile(pw *load.ProgressWriter, fileName string, content io.Reader, contentType string) error {
	slog.Info("Начало установки заголовков", "fileName", fileName)
	pw.Header().Set("Content-Type", contentType)
	pw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	slog.Info("Заголовки установлены")

	slog.Info("Начало передачи данных клиенту", "fileName", fileName)
	_, err := io.Copy(pw, content)
	if err != nil {
		return fmt.Errorf("ошибка при передаче данных: %v", err)
	}
	slog.Info("Данные переданы клиенту", "fileName", fileName)
	return nil
}

func processZip(pw *load.ProgressWriter, object io.ReaderAt, size int64, crc uint32) error {
	slog.Info("Начало обработки ZIP-архива", "размер", size)
	zipReader, err := zip.NewReader(object, size)
	if err != nil {
		return fmt.Errorf("ошибка при чтении ZIP-архива: %v", err)
	}
	slog.Info("Успешно создан читатель ZIP-архива", "количество_файлов", len(zipReader.File))

	searchedFile, err := findSuitableFile(zipReader, crc)
	if err != nil {
		return err
	}
	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc)

	rc, err := searchedFile.Open()
	if err != nil {
		return fmt.Errorf("ошибка при открытии файла в ZIP: %v", err)
	}
	defer rc.Close()
	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

	contentType := getContentType(searchedFile.Name)
	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "тип_содержимого", contentType)

	fileManager := FileHandler(handleFile)
	if err := fileManager(pw, searchedFile.Name, rc, contentType); err != nil {
		return fmt.Errorf("ошибка при обработке файла из ZIP: %v", err)
	}

	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
	return nil
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

func getContentType(fileName string) string {
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

// func generateFileName(contentType string) string {
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

// старый кусок

// func handleRegularFile(w http.ResponseWriter, ctx context.Context, objectID string, object *minio.Object, stat minio.ObjectInfo, startTime time.Time) {
// 	defer object.Close()

// 	fileName := determineFileName(stat)
// 	downloadTime := time.Now()

// 	fileManager := FileHandler(handleFile)
// 	if err := fileManager(w, fileName, object, stat.ContentType); err != nil {
// 		slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("Файл отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
// }

// func handleZipFile(w http.ResponseWriter, ctx context.Context, objectID string, object *minio.Object, stat minio.ObjectInfo, crc32 uint32, size int64, startTime time.Time) {
// 	defer object.Close()

// 	downloadTime := time.Now()

// 	err := processZip(w, object, size, crc32)
// 	if err != nil {
// 		slog.Error("Ошибка обработки ZIP-архива", "object_id", objectID, "error", err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	duration := time.Since(startTime)
// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "duration", duration, "download_time", downloadDuration)
// }

// type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error

// func handleFile(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error {
// 	slog.Info("Начало установки заголовков", "fileName", fileName)
// 	w.Header().Set("Content-Type", contentType)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
// 	slog.Info("Заголовки установлены")

// 	startTime := time.Now()
// 	slog.Info("Начало передачи данных клиенту", "fileName", fileName)
// 	pw := &load.ProgressWriter{Writer: w, LastLogTime: time.Now()}
// 	_, err := io.Copy(pw, content)
// 	if err != nil {
// 		slog.Error("Ошибка при передаче данных", "fileName", fileName, "error", err)
// 		return fmt.Errorf("failed to send file data: %v", err)
// 	}
// 	slog.Info("Данные переданы клиенту", "fileName", fileName, "duration", time.Since(startTime).Seconds(), "total_bytes", pw.Total)
// 	return nil
// }

// func processZip(w http.ResponseWriter, object io.ReaderAt, size int64, crc uint32) error {
// 	slog.Info("Начало обработки ZIP-архива", "размер", size)
// 	zipReader, err := zip.NewReader(object, size)
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

// 	fileManager := FileHandler(handleFile)
// 	if err := fileManager(w, searchedFile.Name, rc, contentType); err != nil {
// 		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return err
// 	}

// 	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
// 	return nil
// }
