package app

import (
	"context"
	"s3_multiclient/config"
	"s3_multiclient/file/minio"
	"s3_multiclient/load"
	"s3_multiclient/server"
)

func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Get()
	if err != nil {
		return err
	}

	minioLoader, err := minio.Init(cfg.MinIO)
	if err != nil {
		return err
	}

	if err = minioLoader.CreateBucket(ctx, cfg.MinIO.Location); err != nil {
		return err
	}

	loader := load.Init(minioLoader)

	server := server.Init(ctx, loader)

	if err := server.Start(cfg.App); err != nil { // тут внутри горутина
		return err
	}

	return nil
}
