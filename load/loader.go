package load

import (
	"context"
)

type FileManager interface {
	UploadFile(ctx context.Context, progressReader *ProgressReader, objectID, contentType, originalFileName string, contentLength int64) error
	DownloadFile(ctx context.Context, pw *ProgressWriter, objectID string, crc32 uint32) error
	DeleteFile(ctx context.Context, objectID string) error
}

type Loader struct {
	fileManager FileManager
}

func Init(fm FileManager) *Loader {
	return &Loader{fileManager: fm}
}
