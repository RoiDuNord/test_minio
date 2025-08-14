package load

import (
	"context"
	"net/http"
)

func (l *Loader) Upload(r *http.Request, ctx context.Context, objectID, contentType, originalFileName string, contentLength int64) error {
	progressReader := newProgressReader(r)

	if err := l.fileManager.UploadFile(ctx, progressReader, objectID, contentType, originalFileName, contentLength); err != nil {
		return err
	}

	return nil
}
