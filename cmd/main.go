package main

import (
	"test_minio/app"
	// "test_minio/handler" для Swagger
)

// @title MinIOClient API
// @version 1.0

// @API Server for MinIO Uploading and Downloading

func main() {
	if err := app.Run(); err != nil {
		return
	}
}
