package main

import (
	"test_minio/server"
)

func main() {
	if err := server.Run(); err != nil {
		return
	}
}
