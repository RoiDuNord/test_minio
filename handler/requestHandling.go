package handler

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

type objectResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

func parseObjectIDandCRC(parsedInfo string) (objectID string, crc uint32, err error) {
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

func getContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	defaultContentType := "application/octet-stream"
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return defaultContentType
}

func parseObjectID(r *http.Request) (string, error) {
	objectID := chi.URLParam(r, "object_id")
	if objectID == "" {
		return "", fmt.Errorf("object_id is required")
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

func sendJSONResponse(w http.ResponseWriter, objectID, originalName, contentType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := &objectResponse{
		ID:     objectID,
		Name:   originalName,
		Type:   contentType,
		Status: successfulUploadStatus,
		// Message: successfulUploadMessage,
		// UploadDuration: uploadDuration.Seconds(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Ошибка формирования JSON ответа", "error", err)
	}
}
