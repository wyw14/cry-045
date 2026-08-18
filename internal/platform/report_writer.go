package platform

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type ReportEncoder func(context.Context, io.Writer) error

type LocalReportWriter struct {
	Root string
}

func (w LocalReportWriter) Write(ctx context.Context, name string, encode ReportEncoder) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(w.Root, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(w.Root, name)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := encode(ctx, file); err != nil {
		return "", err
	}
	return path, nil
}
