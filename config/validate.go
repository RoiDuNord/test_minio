package config

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

func (c *MinIOConfig) Validate() error {
	missingVars := []string{}

	if c.Endpoint == "" {
		missingVars = append(missingVars, "MINIO_ENDPOINT")
	}
	if c.AccessKeyID == "" {
		missingVars = append(missingVars, "MINIO_ACCESS_KEY")
	}
	if c.SecretAccessKey == "" {
		missingVars = append(missingVars, "MINIO_SECRET_KEY")
	}
	if c.BucketName == "" {
		missingVars = append(missingVars, "MINIO_BUCKET_NAME")
	}
	if c.Location == "" {
		missingVars = append(missingVars, "MINIO_LOCATION")
	}

	if len(missingVars) > 0 {
		message := fmt.Sprintf("Необходимо задать переменные окружения: %s", strings.Join(missingVars, ", "))
		slog.Warn(message)
		return errors.New(message)
	}

	if !isValidBucketName(c.BucketName) {
		message := fmt.Sprintf("Имя бакета '%s' содержит недопустимые символы. Используйте только строчные буквы, цифры и дефисы.", c.BucketName)
		slog.Error(message)
		return errors.New(message)
	}

	return nil
}

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9\-]+$`)

func isValidBucketName(bucketName string) bool {
	return bucketNameRegex.MatchString(bucketName)
}

func (c *AppConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("APP_PORT должен быть в диапазоне 1-65535, получено: %d", c.Port)
	}
	return nil
}
