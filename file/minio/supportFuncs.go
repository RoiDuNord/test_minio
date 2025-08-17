package minio

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path/filepath"
	"s3_multiclient/load"
	"s3_multiclient/server"
	"strings"

	"github.com/minio/minio-go/v7"
)

// потом ввести интерфейсы

type downloadedFileData struct {
	metadata    *server.DownloadRequestMetadata
	minioObject minioFileObject
}

type minioFileObject struct {
	reader *minio.Object
	info   minio.ObjectInfo
}

func streamRegularFile(pw *load.ProgressWriter, object *downloadedFileData) error {
	defer object.minioObject.reader.Close() // как изменить имена полей, чтобы они не путались

	fileName := determineFileName(object.minioObject.info)

	fileManager := FileHandler(streamFileContent)
	if err := fileManager(pw, fileName, object.minioObject.reader, object.minioObject.info.ContentType); err != nil {
		return fmt.Errorf("не удалось отправить файл клиенту: %w", err)
	}

	slog.Info("Файл отправлен клиенту", "object_id", object.metadata.ID)
	return nil
}

func streamFileFromZip(pw *load.ProgressWriter, object *downloadedFileData) error {
	defer object.minioObject.reader.Close()

	if err := extractFromZip(pw, object); err != nil {
		return fmt.Errorf("ошибка обработки ZIP-архива: %w", err)
	}

	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", object.metadata.ID)
	return nil
}

type FileHandler func(pw *load.ProgressWriter, fileName string, content io.Reader, contentType string) error

func streamFileContent(pw *load.ProgressWriter, fileName string, content io.Reader, contentType string) error {
	slog.Info("Начало установки заголовков", "file_name", fileName)
	pw.Header().Set("Content-Type", contentType)
	pw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	slog.Info("Заголовки установлены")

	slog.Info("Начало передачи данных клиенту", "file_name", fileName)
	_, err := io.Copy(pw, content)
	if err != nil {
		return fmt.Errorf("ошибка при передаче данных: %v", err)
	}
	slog.Info("Данные переданы клиенту", "file_name", fileName)
	return nil
}

func extractFromZip(pw *load.ProgressWriter, object *downloadedFileData) error {
	slog.Info("Начало обработки ZIP-архива")
	zipReader, err := zip.NewReader(object.minioObject.reader, object.minioObject.info.Size)
	if err != nil {
		return fmt.Errorf("ошибка при чтении ZIP-архива: %v", err)
	}
	slog.Info("Успешно создан читатель ZIP-архива", "files_quantity", len(zipReader.File))

	searchedFile, err := findFileByCRC32(zipReader, object.metadata.CRC32)
	if err != nil {
		return err
	}
	slog.Info("Найден подходящий файл в ZIP-архиве", "file_name", searchedFile.Name, "crc32", object.metadata.CRC32)

	rc, err := searchedFile.Open()
	if err != nil {
		return fmt.Errorf("ошибка при открытии файла в ZIP: %v", err)
	}
	defer rc.Close()
	slog.Info("Файл успешно открыт для чтения", "file_name", searchedFile.Name)

	contentType := getContentType(searchedFile.Name)
	slog.Info("Определен тип содержимого файла", "file_name", searchedFile.Name, "content_type", contentType)

	fileManager := FileHandler(streamFileContent)
	if err := fileManager(pw, searchedFile.Name, rc, contentType); err != nil {
		return fmt.Errorf("ошибка при обработке файла из ZIP: %v", err)
	}

	slog.Info("Обработка ZIP-архива успешно завершена", "file_name", searchedFile.Name)
	return nil
}

func findFileByCRC32(zipReader *zip.Reader, crc32 uint32) (*zip.File, error) {
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
	originalName := stat.UserMetadata[originalNameKey]
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
