package load

import (
	"log/slog"
	"net/http"
	"time"
)

type ProgressWriter struct {
	http.ResponseWriter
	Total       int64
	LastLogTime time.Time
}

func newProgressWriter(w http.ResponseWriter) *ProgressWriter {
	return &ProgressWriter{ResponseWriter: w, LastLogTime: time.Now()}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.ResponseWriter.Write(p)
	if err == nil {
		pw.Total += int64(n)
		now := time.Now()
		if now.Sub(pw.LastLogTime) >= time.Second {
			slog.Info("Прогресс передачи данных", "total_bytes", pw.Total)
			pw.LastLogTime = now
		}
	}
	return n, err
}

type ProgressReader struct {
	*http.Request
	TotalBytes  int64
	ChunkCount  int
	LastLogTime time.Time
}

func newProgressReader(r *http.Request) *ProgressReader {
	return &ProgressReader{
		Request:     r,
		LastLogTime: time.Now(),
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Body.Read(p)
	pr.TotalBytes += int64(n)
	pr.ChunkCount++
	now := time.Now()
	if now.Sub(pr.LastLogTime) >= time.Second {
		slog.Info("Прогресс чтения", "chunk_number", pr.ChunkCount, "bytes_read_in_chunk", n, "total_Mb", pr.TotalBytes/1024/1024)
		pr.LastLogTime = now
	}
	return n, err
}

// Дополнительные методы
func (pw *ProgressWriter) Header() http.Header {
	return pw.ResponseWriter.Header()
}

func (pw *ProgressWriter) WriteHeader(statusCode int) {
	pw.ResponseWriter.WriteHeader(statusCode)
}
