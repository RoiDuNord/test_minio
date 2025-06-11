package object

import (
	"log/slog"
	"os"
	"path/filepath"
)

type Object struct {
	Name        string
	Path        string
	ContentType string
}

func Get() (*Object, error) {
	f := &Object{}

	f.Name = "testdata/1984.jpg"
	relFilePath := "./1984.jpg"
	path, err := getPath(relFilePath)
	if err != nil {
		return nil, err
	}
	f.Path = path
	f.ContentType = "image/jpeg"

	return f, nil
}

func getPath(relFilePath string) (string, error) {
	path, err := filepath.Abs(relFilePath)
	if err != nil {
		slog.Error("Ошибка при получении абсолютного пути к файлу", "error", err)
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Error("Файл не найден", "path", path, "error", err)
		return "", err
	}
	return path, nil
}
