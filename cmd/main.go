package main

import (
	"s3_multiclient/app"
	// "s3_multiclient/fileManager" для Swagger
)

// @title MinIOClient API
// @version 1.0

// @API Server for MinIO Uploading and Downloading

func main() {
	if err := app.Run(); err != nil {
		return
	}
}
