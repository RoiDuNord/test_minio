package config

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

func (mc *MinIOConfig) Validate() error {
	missingVars := []string{}

	if mc.Endpoint == "" {
		missingVars = append(missingVars, "MINIO_ENDPOINT")
	}
	if mc.AccessKeyID == "" {
		missingVars = append(missingVars, "MINIO_ACCESS_KEY")
	}
	if mc.SecretAccessKey == "" {
		missingVars = append(missingVars, "MINIO_SECRET_KEY")
	}
	if mc.BucketName == "" {
		missingVars = append(missingVars, "MINIO_BUCKET_NAME")
	}
	if mc.Location == "" {
		missingVars = append(missingVars, "MINIO_LOCATION")
	}
	if mc.Storage == "" {
		missingVars = append(missingVars, "MINIO_STORAGE")
	}

	if len(missingVars) > 0 {
		message := fmt.Sprintf("Необходимо задать переменные окружения: %s", strings.Join(missingVars, ", "))
		slog.Warn(message)
		return errors.New(message)
	}

	if !isValidBucketName(mc.BucketName) {
		message := fmt.Sprintf("Имя бакета '%s' содержит недопустимые символы. Используйте только строчные буквы, цифры и дефисы.", mc.BucketName)
		slog.Error(message)
		return errors.New(message)
	}

	return nil
}

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9\-]+$`)

func isValidBucketName(bucketName string) bool {
	return bucketNameRegex.MatchString(bucketName)
}

func (ap *AppConfig) Validate() error {
	if ap.Port <= 0 || ap.Port > 65535 {
		return fmt.Errorf("APP_PORT должен быть в диапазоне 1-65535, получено: %d", ap.Port)
	}
	return nil
}
