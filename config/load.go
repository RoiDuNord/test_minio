package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

func (c *MinIOConfig) Load(envMap map[string]string) error {
	var ok bool

	c.Endpoint, ok = envMap["MINIO_ENDPOINT"]
	if !ok {
		slog.Warn("Переменная MINIO_ENDPOINT не определена в .env")
	}

	c.AccessKeyID, ok = envMap["MINIO_ACCESS_KEY"]
	if !ok {
		slog.Warn("Переменная MINIO_ACCESS_KEY не определена в .env")
	}

	c.SecretAccessKey, ok = envMap["MINIO_SECRET_KEY"]
	if !ok {
		slog.Warn("Переменная MINIO_SECRET_KEY не определена в .env")
	}

	useSSLStr, ok := envMap["MINIO_USE_SSL"]
	if ok {
		c.UseSSL = strings.ToLower(useSSLStr) != "false"
	} else {
		c.UseSSL = true
		slog.Warn("Переменная MINIO_USE_SSL не определена в .env, используется значение по умолчанию: true")
	}

	c.BucketName, ok = envMap["MINIO_BUCKET_NAME"]
	if !ok {
		slog.Warn("Переменная MINIO_BUCKET_NAME не определена в .env")
	}

	c.Location, ok = envMap["MINIO_LOCATION"]
	if !ok {
		slog.Warn("Переменная MINIO_LOCATION не определена в .env")
	}

	return nil
}

func (c *AppConfig) Load(envMap map[string]string) error {
	portStr, ok := envMap["APP_PORT"]
	if !ok {
		slog.Warn("Переменная APP_PORT не определена в .env")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("ошибка преобразования APP_PORT в число: %w", err)
	}

	c.Port = port
	return nil
}
