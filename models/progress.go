package models

import (
	"io"
	"log/slog"
	"time"
)

type ProgressReader struct {
	Reader      io.Reader
	TotalBytes  int64
	ChunkCount  int
	LastLogTime time.Time
}

func NewProgressReader(r io.Reader) *ProgressReader {
	return &ProgressReader{Reader: r, LastLogTime: time.Now()}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.TotalBytes += int64(n)
	pr.ChunkCount++
	now := time.Now()
	if now.Sub(pr.LastLogTime) >= time.Second {
		slog.Info("Прогресс чтения", "chunk_number", pr.ChunkCount, "bytes_read_in_chunk", n, "total_Mb", pr.TotalBytes/1024/1024)
		pr.LastLogTime = now
	}
	return n, err
}

type ProgressWriter struct {
	Writer      io.Writer
	Total       int64
	LastLogTime time.Time
}

func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{Writer: w, LastLogTime: time.Now()}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Total += int64(n)
	now := time.Now()
	if now.Sub(pw.LastLogTime) >= time.Second {
		slog.Info("Прогресс передачи данных", "total_bytes", pw.Total)
		pw.LastLogTime = now
	}
	return n, err
}
