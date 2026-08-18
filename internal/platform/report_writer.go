package platform

import (
	"context"
	"errors"
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
	if encode == nil {
		return "", errors.New("report encoder is required")
	}
	root, finalPath, err := w.paths(name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", fmt.Errorf("create report directory: %w", err)
	}

	temporary, err := os.CreateTemp(root, ".report-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary report: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := encode(ctx, temporary); err != nil {
		return "", fmt.Errorf("encode report: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		return "", fmt.Errorf("sync report: %w", err)
	}
	if err := temporary.Chmod(0o640); err != nil {
		return "", fmt.Errorf("set report permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close report: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := replaceFile(temporaryPath, finalPath); err != nil {
		return "", err
	}
	committed = true
	return finalPath, nil
}

func (w LocalReportWriter) paths(name string) (string, string, error) {
	root := filepath.Clean(strings.TrimSpace(w.Root))
	if root == "." || root == "" {
		return "", "", errors.New("report root is required")
	}
	cleanName := filepath.Base(strings.TrimSpace(name))
	if cleanName == "." || cleanName == "" || cleanName != name || strings.Contains(name, "..") {
		return "", "", errors.New("invalid report name")
	}
	if strings.ToLower(filepath.Ext(cleanName)) != ".json" {
		return "", "", errors.New("report must use .json extension")
	}
	finalPath := filepath.Join(root, cleanName)
	return root, finalPath, nil
}

func replaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	}
	backup := destination + ".previous"
	_ = os.Remove(backup)
	if err := os.Rename(destination, backup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("prepare report replacement: %w", err)
	}
	if err := os.Rename(source, destination); err != nil {
		_ = os.Rename(backup, destination)
		return fmt.Errorf("replace report: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}
