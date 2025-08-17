package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
)

const (
	defaultUploadFileName  = "default_name.bin"
	successfulUploadStatus = "uploaded"
)

func getIDandCRC(parsedData string) (handledData *DownloadRequestMetadata, err error) {
	parsedData = strings.TrimSpace(parsedData)
	if parsedData == "" {
		err = fmt.Errorf("object identifier is required")
		slog.Error("Пустой идентификатор объекта", "error", err)
		return nil, err
	}

	parts := strings.Split(parsedData, ";")
	if len(parts) > 2 {
		err = fmt.Errorf("invalid format: expected 'objectID;crc32'")
		slog.Error("Неверный формат идентификатора", "error", err, "parsed_info", parsedData)
		return nil, err
	}

	objectID := strings.TrimSpace(parts[0])
	if objectID == "" {
		err = fmt.Errorf("object identifier is required")
		slog.Error("Пустой идентификатор объекта после разделения", "error", err, "parsed_info", parsedData)
		return nil, err
	}

	var crc32 uint32
	if len(parts) == 2 {
		crcStr := strings.TrimSpace(parts[1])
		crcValue, parseErr := strconv.ParseUint(crcStr, 10, 32)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse CRC value: %v", parseErr)
			slog.Error("Ошибка разбора значения CRC", "error", err, "crc32", crcStr)
			return nil, err
		}
		crc32 = uint32(crcValue)
	}

	handledData = &DownloadRequestMetadata{ID: objectID, CRC32: crc32}

	return handledData, nil
}

func getContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	defaultContentType := "application/octet-stream"
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return defaultContentType
}

func getUploadRequestData(r *http.Request) (*UploadRequestMetadata, error) {
	objectID, err := parseObjectID(r)
	if err != nil {
		slog.Error("Не удалось извлечь object_id", "error", err)
		return nil, err
	}

	fileName, err := parseFileNameFromDisposition(r)
	if err != nil {
		slog.Warn("Не удалось извлечь имя файла", "error", err)
		return nil, err
	}

	contentType := getContentType(fileName)
	slog.Info("Определен тип содержимого файла", "file_name", fileName, "content_type", contentType)

	contentLength := r.ContentLength

	data := &UploadRequestMetadata{
		ID:          objectID,
		FileName:    fileName,
		ContentType: contentType,
		Size:        contentLength,
	}

	return data, nil
}

func getSizeMB(size int64) int {
	return int(size / (1024 * 1024))
}

func parseObjectID(r *http.Request) (string, error) {
	objectID := chi.URLParam(r, "object_id")
	if objectID == "" {
		return "", fmt.Errorf("необходим object_id")
	}
	return objectID, nil
}

func parseFileNameFromDisposition(r *http.Request) (string, error) {
	originalName := defaultUploadFileName
	if contentDisposition := r.Header.Get("Content-Disposition"); contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				name = filepath.Base(name)
				name = strings.ReplaceAll(name, "..", "")
				if name != "" {
					originalName = name
					return originalName, nil
				}
			}
		} else {
			slog.Warn("Ошибка разбора Content-Disposition", "error", err)
		}
	}
	return originalName, nil
}

func sendJSONResponse(w http.ResponseWriter, data *UploadRequestMetadata) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	size := getSizeMB(data.Size)

	response := &objectResponse{
		Status: successfulUploadStatus,
		ID:     data.ID,
		Name:   data.FileName,
		Type:   data.ContentType,
		Size:   size,
		// Message: successfulUploadMessage,
		// UploadDuration: uploadDuration.Seconds(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Ошибка формирования JSON ответа", "error", err)
	}
}
