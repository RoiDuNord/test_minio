package models

import (
	"io"
	"log/slog"
	"time"
)

type ProgressReader struct {
	R          io.Reader
	TotalBytes int
	ChunkCount int
	Last       time.Time
}

func NewProgressReader(r io.Reader) *ProgressReader {
	return &ProgressReader{R: r, Last: time.Now()}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.R.Read(p)
	pr.TotalBytes += n
	pr.ChunkCount++
	now := time.Now()
	if now.Sub(pr.Last) >= time.Second {
		slog.Info("Прогресс чтения", "chunk_number", pr.ChunkCount, "bytes_read_in_chunk", n, "total_Mb", pr.TotalBytes/1024/1024)
		pr.Last = now
	}
	return n, err
}

type ProgressWriter struct {
	W     io.Writer
	Total int64
	Last  time.Time
}

func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{W: w, Last: time.Now()}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.W.Write(p)
	pw.Total += int64(n)
	now := time.Now()
	if now.Sub(pw.Last) >= time.Second {
		slog.Info("Прогресс передачи данных", "total_bytes", pw.Total)
		pw.Last = now
	}
	return n, err
}
