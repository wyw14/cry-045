package platform

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFailedReportExportPreservesExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "selection.json")
	if err := os.WriteFile(path, []byte("stable-report"), 0o640); err != nil {
		t.Fatal(err)
	}
	writer := LocalReportWriter{Root: root}
	_, err := writer.Write(context.Background(), "selection.json", func(_ context.Context, output io.Writer) error {
		_, _ = io.WriteString(output, "partial-report")
		return errors.New("encoder failed")
	})
	if err == nil {
		t.Fatal("Write succeeded, want encoder failure")
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "stable-report" {
		t.Fatalf("existing report was corrupted: %q", content)
	}
	matches, globErr := filepath.Glob(filepath.Join(root, ".report-*.tmp"))
	if globErr != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v, err=%v", matches, globErr)
	}
}
