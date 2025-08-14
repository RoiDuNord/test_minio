package app

import (
	"context"

	"s3_multiclient/config"
	"s3_multiclient/file/minio"
	"s3_multiclient/handler"
	"s3_multiclient/load"
)

func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Get()
	if err != nil {
		return err
	}

	minio, err := minio.Init(cfg.MinIO)
	if err != nil {
		return err
	}

	if err = minio.CreateBucket(ctx, cfg.MinIO.Location); err != nil {
		return err
	}

	loader := load.Init(minio)

	server := handler.NewServer(ctx, loader)

	if err := server.Start(cfg.App); err != nil { // тут внутри горутина
		return err
	}

	return nil
}
