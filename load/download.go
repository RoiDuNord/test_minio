package load

import (
	"context"
	"net/http"
	"s3_multiclient/server"
)

func (l *Loader) Download(w http.ResponseWriter, ctx context.Context, data *server.DownloadRequestMetadata) error {
	pw := newProgressWriter(w)

	if err := l.fileManager.DownloadFile(ctx, pw, data); err != nil {
		return err
	}
	return nil
}
