package db

import (
	"context"
	"net/http"
)

type DBManager interface {
	UploadInfo(r *http.Request, ctx context.Context, objectID, contentType, originalName string) error
	DownloadInfo(w http.ResponseWriter, ctx context.Context, objectID string, crc uint32) error
	DeleteInfo(objectID string) error
}
