package config

import (
	"fmt"
	"log/slog"
	"strconv"
)

func (mc *MinIOConfig) Load(envMap map[string]string) error {
	var ok bool
	var missingVars []string

	mc.Endpoint, ok = envMap["MINIO_ENDPOINT"]
	if !ok {
		missingVars = append(missingVars, "MINIO_ENDPOINT")
	}

	mc.AccessKeyID, ok = envMap["MINIO_ACCESS_KEY"]
	if !ok {
		missingVars = append(missingVars, "MINIO_ACCESS_KEY")
	}

	mc.SecretAccessKey, ok = envMap["MINIO_SECRET_KEY"]
	if !ok {
		missingVars = append(missingVars, "MINIO_SECRET_KEY")
	}

	mc.BucketName, ok = envMap["MINIO_BUCKET_NAME"]
	if !ok {
		missingVars = append(missingVars, "MINIO_BUCKET_NAME")
	}

	mc.Location, ok = envMap["MINIO_LOCATION"]
	if !ok {
		missingVars = append(missingVars, "MINIO_LOCATION")
	}

	mc.Storage, ok = envMap["MINIO_STORAGE"]
	if !ok {
		missingVars = append(missingVars, "MINIO_STORAGE")
	}

	useSSLStr, ok := envMap["MINIO_USE_SSL"]
	if !ok {
		missingVars = append(missingVars, "MINIO_USE_SSL")
	} else {
		mc.UseSSL = (useSSLStr == "true")
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			slog.Warn(fmt.Sprintf("Переменная %s не определена в .env", v))
		}
		return fmt.Errorf("отсутствуют обязательные переменные окружения MINIO: %v", missingVars)
	}

	return nil
}

func (ap *AppConfig) Load(envMap map[string]string) error {
	var ok bool
	var missingVars []string

	ap.Host, ok = envMap["APP_HOST"]
	if !ok {
		missingVars = append(missingVars, "APP_HOST")
	}

	if err := ap.loadPort(envMap); err != nil {
		return err
	}

	if len(missingVars) > 0 {
		for _, v := range missingVars {
			slog.Warn(fmt.Sprintf("Переменная %s не определена в .env", v))
		}
		return fmt.Errorf("отсутствуют обязательные переменные окружения APP: %v", missingVars)
	}

	return nil
}

func (ap *AppConfig) loadPort(envMap map[string]string) error {
	portStr, ok := envMap["APP_PORT"]
	if !ok {
		slog.Warn("Переменная APP_PORT не определена в .env")
		return fmt.Errorf("переменная APP_PORT не определена в .env")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("ошибка преобразования APP_PORT в число: %w", err)
	}

	ap.Port = port
	return nil
}
