package load

import (
	"context"
	"s3_multiclient/server"
)

type FileManager interface {
	UploadFile(ctx context.Context, progressReader *ProgressReader, data *server.UploadRequestMetadata) error
	DownloadFile(ctx context.Context, pw *ProgressWriter, data *server.DownloadRequestMetadata) error
	DeleteFile(ctx context.Context, objectID string) error
}

type Loader struct {
	fileManager FileManager
}

func Init(fm FileManager) *Loader {
	return &Loader{fileManager: fm}
}
