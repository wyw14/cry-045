package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ReportEncoder func(context.Context, io.Writer) error

type LocalReportWriter struct {
	Root string
}

func (w LocalReportWriter) Write(ctx context.Context, name string, encode ReportEncoder) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	base := filepath.Base(name)
	if base != name || strings.Contains(name, "..") || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("invalid report name")
	}
	if strings.ToLower(filepath.Ext(base)) != ".json" {
		return "", fmt.Errorf("report must use .json extension")
	}
	if err := os.MkdirAll(w.Root, 0o750); err != nil {
		return "", err
	}
	target := filepath.Join(w.Root, name)
	tmp, err := os.CreateTemp(w.Root, ".report-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if encodeErr := encode(ctx, tmp); encodeErr != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", encodeErr
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return target, nil
}
