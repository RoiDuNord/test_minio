package load

import (
	"context"
	"net/http"
	"s3_multiclient/server"
)

func (l *Loader) Upload(r *http.Request, ctx context.Context, data *server.UploadRequestMetadata) error {
	progressReader := newProgressReader(r)

	if err := l.fileManager.UploadFile(ctx, progressReader, data); err != nil {
		return err
	}

	return nil
}
