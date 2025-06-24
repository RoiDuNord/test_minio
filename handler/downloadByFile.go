// package handler

// import (
// 	"archive/zip"
// 	"os"
// 	"strconv"
// 	"sync"

// 	"context"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"mime"
// 	"net/http"
// 	"path/filepath"
// 	"strings"
// 	"time"

// 	"github.com/go-chi/chi"
// 	"github.com/minio/minio-go/v7"
// 	"golang.org/x/sync/semaphore"
// )

// const (
// 	chunkSize = 5 * 1024 * 1024
// )

// func (s *Server) Download(w http.ResponseWriter, r *http.Request) {
// 	if err := checkGetMethod(r); err != nil {
// 		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
// 		slog.Error("Недопустимый метод запроса", "error", err)
// 		return
// 	}

// 	startTime := time.Now()
// 	objectID, crc, err := parseObjectIDandCRC(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	slog.Info("Начало обработки запроса на скачивание", "object_id", objectID)

// 	stat, err := s.getObjectStat(s.Ctx, objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить метаданные объекта", "object_id", objectID, "error", err)
// 		http.Error(w, "File not found", http.StatusNotFound)
// 		return
// 	}

// 	if stat.ContentType == "application/zip" {
// 		if crc == 0 {
// 			slog.Error("Отсутствует CRC для ZIP", "object_id", objectID)
// 			http.Error(w, "No CRC for ZIP", http.StatusBadRequest)
// 			return
// 		}
// 		s.handleZipFile(w, r, objectID, crc, stat.Size, startTime)
// 		return
// 	}
// 	s.handleRegularFile(w, r, objectID, stat, startTime)
// }

// type FileHandler func(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error

// func checkGetMethod(r *http.Request) error {
// 	if r.Method != http.MethodGet {
// 		return fmt.Errorf("method not allowed: %s", r.Method)
// 	}
// 	return nil
// }

// func parseObjectIDandCRC(r *http.Request) (objectID string, crc32 uint32, err error) {
// 	objectID = chi.URLParam(r, "object_id")
// 	if objectID == "" {
// 		err = fmt.Errorf("object identifier is required")
// 		slog.Error("Пустой идентификатор объекта", "error", err)
// 		return "", 0, err
// 	}

// 	if strings.Contains(objectID, ";") {
// 		parts := strings.Split(objectID, ";")
// 		if len(parts) != 2 {
// 			err = fmt.Errorf("invalid format: expected 'fileID;crc' for ZIP archive")
// 			slog.Error("Неверный формат идентификатора для ZIP", "error", err, "input", objectID)
// 			return "", 0, err
// 		}

// 		objectID = parts[0]
// 		if objectID == "" {
// 			err = fmt.Errorf("object identifier is required")
// 			slog.Error("Пустой идентификатор объекта в формате ZIP", "error", err, "input", objectID)
// 			return "", 0, err
// 		}

// 		crcValue, err := strconv.ParseUint(parts[1], 10, 32)
// 		if err != nil {
// 			err = fmt.Errorf("failed to parse CRC value: %v", err)
// 			slog.Error("Ошибка разбора значения CRC", "error", err, "input", parts[1])
// 			return "", 0, err
// 		}

// 		return objectID, uint32(crcValue), nil
// 	}

// 	return objectID, 0, nil
// }

// func getContentType(fileName string) string {
// 	ext := strings.ToLower(filepath.Ext(fileName))
// 	defaultContentType := "application/octet-stream"
// 	if contentType := mime.TypeByExtension(ext); contentType != "" {
// 		return contentType
// 	}
// 	return defaultContentType
// }

// type ProgressWriter struct {
// 	w          io.Writer
// 	last       time.Time
// 	totalBytes int64
// }

// // Write реализует метод Write для ProgressWriter
// func (pw *ProgressWriter) Write(p []byte) (int, error) {
// 	n, err := pw.w.Write(p)
// 	if err != nil {
// 		return n, err
// 	}
// 	pw.totalBytes += int64(n)
// 	now := time.Now()
// 	if time.Since(pw.last) > 1*time.Millisecond {
// 		slog.Info("Прогресс записи данных", "bytes_written", n, "total_bytes", pw.totalBytes)
// 		pw.last = now
// 	}
// 	return n, nil
// }

// func handleFile(w http.ResponseWriter, fileName string, content io.Reader, contentType string) error {
// 	start := time.Now()
// 	slog.Info("Начало установки заголовков", "fileName", fileName)
// 	w.Header().Set("Content-Type", contentType)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
// 	slog.Info("Заголовки установлены", "duration", time.Since(start).Seconds())

// 	start = time.Now()
// 	slog.Info("Начало передачи данных клиенту", "fileName", fileName)
// 	pw := &ProgressWriter{w: w, last: time.Now()}
// 	_, err := io.Copy(pw, content)
// 	if err != nil {
// 		slog.Error("Ошибка при передаче данных", "fileName", fileName, "error", err)
// 		return fmt.Errorf("failed to send file data: %v", err)
// 	}
// 	slog.Info("Данные переданы клиенту", "fileName", fileName, "duration", time.Since(start).Seconds(), "total_bytes", pw.totalBytes)
// 	return nil
// }

// // findSuitableFile ищет файл в ZIP-архиве по CRC32
// func findSuitableFile(zipReader *zip.Reader, crc32 uint32) (*zip.File, error) {
// 	for _, file := range zipReader.File {
// 		fmt.Println("crc32 of file", file.CRC32)
// 		if file.CRC32 == crc32 {
// 			return file, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("file with CRC32 %d not found in ZIP archive", crc32)
// }

// func getChunksQuantity(fileSize int64) int {
// 	totalParts := (fileSize + chunkSize - 1) / chunkSize
// 	slog.Info("fileSize and totalParts", "fileSize", fileSize, "totalParts", totalParts)
// 	return int(totalParts)
// }

// func (s *Server) getObjectStat(ctx context.Context, objectID string) (minio.ObjectInfo, error) {
// 	return s.MinioClient.Client.StatObject(ctx, s.MinioClient.BucketName, objectID, minio.StatObjectOptions{})
// }

// func (s *Server) getObject(ctx context.Context, objectID string) (*minio.Object, error) {
// 	return s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectID, minio.GetObjectOptions{})
// }

// func (s *Server) determineFileName(stat minio.ObjectInfo) string {
// 	originalName := stat.UserMetadata["X-Original-Name"]
// 	if originalName != "" {
// 		return originalName
// 	}
// 	return stat.Key
// }

// type partResult struct {
// 	partNumber int
// 	filePath   string
// 	err        error
// }

// func (s *Server) downloadFilePart(ctx context.Context, bucket, object string, partNumber int, startByte, endByte int64, wg *sync.WaitGroup, ch chan<- partResult) {
// 	defer wg.Done()

// 	// Создаем временный файл для сохранения части
// 	path := "./results"
// 	filePath := filepath.Join(path, fmt.Sprintf("part-%s-%d", object[:3], partNumber))

// 	// Настраиваем опции для скачивания части с использованием Range-запроса
// 	opts := minio.GetObjectOptions{}
// 	opts.SetRange(startByte, endByte)

// 	// Скачиваем часть и сохраняем в файл
// 	err := s.MinioClient.Client.FGetObject(ctx, bucket, object, filePath, opts)
// 	if err != nil {
// 		slog.Error("Ошибка при скачивании части", "part", partNumber, "object", object, "error", err)
// 		ch <- partResult{partNumber: partNumber, filePath: "", err: err}
// 		return
// 	}

// 	slog.Info("Часть файла успешно скачана", "part", partNumber, "object", object, "filePath", filePath)
// 	ch <- partResult{partNumber: partNumber, filePath: filePath, err: nil}
// }

// func (s *Server) mergeAndSendParts(w http.ResponseWriter, fileName string, contentType string, totalParts int, objectID string) error {
// 	// Настраиваем заголовки ответа
// 	w.Header().Set("Content-Type", contentType)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))

// 	// Создаем ProgressWriter для отслеживания прогресса
// 	pw := &ProgressWriter{w: w, last: time.Now()}

// 	// Объединяем части в порядке номеров
// 	for i := 1; i <= totalParts; i++ {
// 		path := "./results"
// 		partPath := filepath.Join(path, fmt.Sprintf("part-%s-%d", objectID[:3], i)) // Используем objectID вместо fileName
// 		partData, err := os.ReadFile(partPath)
// 		if err != nil {
// 			slog.Error("Ошибка при чтении части файла", "part", i, "path", partPath, "error", err)
// 			return fmt.Errorf("ошибка при чтении части %d: %v", i, err)
// 		}

// 		_, err = pw.Write(partData)
// 		if err != nil {
// 			slog.Error("Ошибка при записи части клиенту", "part", i, "error", err)
// 			return fmt.Errorf("ошибка при записи части %d: %v", i, err)
// 		}

// 		// Удаляем временный файл после использования
// 		if err := os.Remove(partPath); err != nil {
// 			slog.Warn("Не удалось удалить временный файл части", "part", i, "path", partPath, "error", err)
// 		}
// 	}

// 	slog.Info("Все части файла успешно объединены и отправлены клиенту", "fileName", fileName, "totalBytes", pw.totalBytes)
// 	return nil
// }

// // Обновленная функция handleRegularFile с поддержкой параллельного скачивания
// func (s *Server) handleRegularFile(w http.ResponseWriter, r *http.Request, objectID string, stat minio.ObjectInfo, startTime time.Time) {
// 	fileName := s.determineFileName(stat)
// 	downloadTime := time.Now()

// 	// Вычисляем количество частей на основе размера файла
// 	totalParts := getChunksQuantity(stat.Size)
// 	slog.Info("Количество частей для скачивания", "object_id", objectID, "totalParts", totalParts)

// 	// Если файл маленький, скачиваем его целиком без разделения на части
// 	if totalParts == 1 {
// 		object, err := s.getObject(s.Ctx, objectID)
// 		if err != nil {
// 			slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
// 			http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 			return
// 		}
// 		defer object.Close()

// 		handler := FileHandler(handleFile)
// 		if err := handler(w, fileName, object, stat.ContentType); err != nil {
// 			slog.Error("Не удалось отправить файл клиенту", "object_id", objectID, "error", err)
// 			http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 			return
// 		}

// 		downloadDuration := time.Since(downloadTime)
// 		slog.Info("Файл отправлен клиенту (без разделения на части)", "object_id", objectID, "download_time", downloadDuration)
// 		return
// 	}

// 	// Создаем канал для получения результатов скачивания
// 	ch := make(chan partResult, totalParts)

// 	// Создаем семафор с ограничением, например, 4 параллельных скачивания
// 	maxConcurrent := int64(4) // Можно настроить в зависимости от ваших требований
// 	sem := semaphore.NewWeighted(maxConcurrent)

// 	// Запускаем параллельное скачивание частей с использованием семафора
// 	var wg sync.WaitGroup
// 	for partNumber := 1; partNumber <= totalParts; partNumber++ {
// 		// Запрашиваем "слот" у семафора перед запуском горутины
// 		if err := sem.Acquire(s.Ctx, 1); err != nil {
// 			slog.Error("Ошибка при получении семафора", "part", partNumber, "error", err)
// 			http.Error(w, fmt.Sprintf("Semaphore error for part %d", partNumber), http.StatusInternalServerError)
// 			return
// 		}

// 		wg.Add(1)
// 		// Вычисляем диапазон байтов для Range-запроса
// 		startByte := int64((partNumber - 1)) * chunkSize
// 		endByte := startByte + chunkSize - 1
// 		if endByte >= stat.Size {
// 			endByte = stat.Size - 1
// 		}
// 		go func(partNum int, start, end int64) {
// 			defer sem.Release(1) // Освобождаем слот после завершения
// 			s.downloadFilePart(s.Ctx, s.MinioClient.BucketName, objectID, partNum, start, end, &wg, ch)
// 		}(partNumber, startByte, endByte)
// 	}

// 	// Ждем завершения всех горутин скачивания
// 	wg.Wait()
// 	close(ch)

// 	// Проверяем результаты скачивания
// 	results := make(map[int]partResult)
// 	for result := range ch {
// 		results[result.partNumber] = result
// 	}

// 	// Проверяем, все ли части скачаны успешно
// 	for i := 1; i <= totalParts; i++ {
// 		if result, ok := results[i]; !ok || result.err != nil {
// 			slog.Error("Ошибка при скачивании части", "part", i, "object_id", objectID, "error", result.err)
// 			http.Error(w, fmt.Sprintf("Failed to download part %d", i), http.StatusInternalServerError)
// 			return
// 		}
// 	}

// 	// Объединяем части и отправляем клиенту
// 	err := s.mergeAndSendParts(w, fileName, stat.ContentType, totalParts, objectID)
// 	if err != nil {
// 		slog.Error("Ошибка при объединении и отправке файла", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return
// 	}

// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("Файл отправлен клиенту", "object_id", objectID, "download_time", downloadDuration)
// }

// func (s *Server) handleZipFile(w http.ResponseWriter, r *http.Request, objectID string, crc uint32, size int64, startTime time.Time) {
// 	downloadTime := time.Now()
// 	// object, err := s.getObject(s.Ctx, objectID, 0)
// 	object, err := s.getObject(s.Ctx, objectID)
// 	if err != nil {
// 		slog.Error("Не удалось получить объект из MinIO", "object_id", objectID, "error", err)
// 		http.Error(w, "Failed to download file", http.StatusInternalServerError)
// 		return
// 	}
// 	defer object.Close()

// 	err = processZip(w, r, object, size, crc)
// 	if err != nil {
// 		slog.Error("Ошибка обработки ZIP-архива", "object_id", objectID, "error", err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	downloadDuration := time.Since(downloadTime)
// 	slog.Info("ZIP-архив обработан и отправлен клиенту", "object_id", objectID, "download_time", downloadDuration)
// }

// // через буфер
// // func processZip(w http.ResponseWriter, r *http.Request, object *minio.Object, size int64, crc32 uint32) error {
// // 	slog.Info("Начало обработки ZIP-архива", "размер", size)

// // 	buf := new(bytes.Buffer)
// // 	_, err := io.Copy(buf, object)
// // 	if err != nil {
// // 		return fmt.Errorf("не удалось прочитать данные ZIP: %v", err)
// // 	}

// // 	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), size)
// // 	if err != nil {
// // 		return fmt.Errorf("не удалось прочитать ZIP-архив: %v", err)
// // 	}

// // 	searchedFile, err := findSuitableFile(zipReader, crc32)
// // 	if err != nil {
// // 		slog.Error("Не удалось найти подходящий файл в ZIP", "error", err, "crc32", crc32)
// // 		return err
// // 	}
// // 	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc32)

// // 	rc, err := searchedFile.Open()
// // 	if err != nil {
// // 		slog.Error("Ошибка при открытии файла в ZIP", "file", searchedFile.Name, "error", err)
// // 		return fmt.Errorf("failed to open file %s in ZIP: %v", searchedFile.Name, err)
// // 	}
// // 	defer rc.Close()
// // 	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

// // 	contentType := getContentType(searchedFile.Name)
// // 	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "contentType", contentType)

// // 	handler := FileHandler(handleFile)
// // 	if err := handler(w, searchedFile.Name, rc, contentType); err != nil {
// // 		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
// // 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// // 		return err
// // 	}

// // 	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
// // 	return nil
// // }

// // через файл
// func processZip(w http.ResponseWriter, r *http.Request, object *minio.Object, size int64, crc32 uint32) error {
// 	slog.Info("Начало обработки ZIP-архива", "размер", size)

// 	// Создаем временный файл
// 	tmpFile, err := os.CreateTemp("", "zip-archive-*.zip")
// 	if err != nil {
// 		return fmt.Errorf("не удалось создать временный файл: %v", err)
// 	}
// 	defer os.Remove(tmpFile.Name())
// 	defer tmpFile.Close()

// 	// Замеряем время чтения из MinIO
// 	start := time.Now()
// 	progressReader := &ProgressReader{r: object, last: time.Now()}
// 	_, err = io.Copy(tmpFile, progressReader)
// 	if err != nil {
// 		return fmt.Errorf("не удалось записать данные во временный файл: %v", err)
// 	}
// 	slog.Info("Данные ZIP-архива прочитаны из MinIO", "duration", time.Since(start).Seconds(), "size_mb", size/1024/1024)

// 	// Открываем временный файл для чтения ZIP
// 	zipReader, err := zip.OpenReader(tmpFile.Name())
// 	if err != nil {
// 		return fmt.Errorf("не удалось прочитать ZIP-архив: %v", err)
// 	}
// 	defer zipReader.Close()

// 	// Поиск файла и дальнейшая обработка
// 	searchedFile, err := findSuitableFile(&zipReader.Reader, crc32) // Используем &zipReader.Reader вместо zipReader
// 	if err != nil {
// 		slog.Error("Не удалось найти подходящий файл в ZIP", "error", err, "crc32", crc32)
// 		return err
// 	}
// 	slog.Info("Найден подходящий файл в ZIP-архиве", "имя_файла", searchedFile.Name, "crc32", crc32)

// 	rc, err := searchedFile.Open()
// 	if err != nil {
// 		slog.Error("Ошибка при открытии файла в ZIP", "file", searchedFile.Name, "error", err)
// 		return fmt.Errorf("failed to open file %s in ZIP: %v", searchedFile.Name, err)
// 	}
// 	defer rc.Close()
// 	slog.Info("Файл успешно открыт для чтения", "имя_файла", searchedFile.Name)

// 	contentType := getContentType(searchedFile.Name)
// 	slog.Info("Определен тип содержимого файла", "имя_файла", searchedFile.Name, "contentType", contentType)

// 	handler := FileHandler(handleFile)
// 	if err := handler(w, searchedFile.Name, rc, contentType); err != nil {
// 		slog.Error("Ошибка при обработке файла из ZIP", "file", searchedFile.Name, "error", err)
// 		http.Error(w, "Failed to send file data", http.StatusInternalServerError)
// 		return err
// 	}

// 	slog.Info("Обработка ZIP-архива успешно завершена", "имя_файла", searchedFile.Name)
// 	return nil
// }

package handler
