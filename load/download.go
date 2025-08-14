package load

import (
	"context"
	"net/http"
)

func (l *Loader) Download(w http.ResponseWriter, ctx context.Context, objectID string, crc32 uint32) error {
	pw := newProgressWriter(w)

	if err := l.fileManager.DownloadFile(ctx, pw, objectID, crc32); err != nil {
		return err
	}
	return nil
}
