package config

import (
	"fmt"
	"log/slog"

	"github.com/joho/godotenv"
)

type BasicConfig interface {
	Load(map[string]string) error
	Validate() error
}

type AppConfig struct {
	Host string
	Port int
}

type MinIOConfig struct {
	UseSSL          bool
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Location        string
	Storage         string
}

type Config struct {
	App   AppConfig
	MinIO MinIOConfig
}

func readEnv() (map[string]string, error) {
	myEnv, err := godotenv.Read()
	if err != nil {
		slog.Error("Ошибка при чтении переменных окружения", "error", err)
		return nil, fmt.Errorf("ошибка при чтении переменных окружения: %w", err)
	}
	return myEnv, nil
}

func Get() (Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Error("Ошибка при загрузке .env файла", "error", err)
		return Config{}, fmt.Errorf("ошибка при загрузке .env файла: %w", err)
	}

	envMap, err := readEnv()
	if err != nil {
		return Config{}, fmt.Errorf("ошибка при чтении переменных окружения: %w", err)
	}

	appCfg := &AppConfig{}
	minioCfg := &MinIOConfig{}

	configs := []BasicConfig{appCfg, minioCfg}
	for _, cfg := range configs {
		if err := cfg.Load(envMap); err != nil {
			slog.Error("Ошибка при загрузке конфигурации", "error", err)
			return Config{}, err
		}
		if err := cfg.Validate(); err != nil {
			slog.Error("Ошибка валидации конфигурации", "error", err)
			return Config{}, err
		}
	}

	slog.Info("Все конфигурации успешно загружены")
	return Config{
		App:   *appCfg,
		MinIO: *minioCfg,
	}, nil
}
