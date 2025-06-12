package config

import (
	"fmt"
	"log/slog"

	"github.com/joho/godotenv"
)

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	Location        string
}

func Get() (*MinIOConfig, error) {
	cfg := &MinIOConfig{}

	if err := cfg.Load(); err != nil {
		slog.Error("Ошибка при загрузке конфигурации из .env", "error", err)
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("Ошибка валидации конфигурации из .env", "error", err)
		return nil, err
	}

	slog.Info("config успешно загружен")
	return cfg, nil
}

func readEnv() (map[string]string, error) {
	myEnv, err := godotenv.Read()
	if err != nil {
		slog.Error("Ошибка при чтении переменных окружения", "error", err)
		return nil, fmt.Errorf("ошибка при чтении переменных окружения: %w", err)
	}
	return myEnv, nil
}
