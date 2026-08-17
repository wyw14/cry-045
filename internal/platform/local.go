package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalAttachmentStore struct {
	Root     string
	MaxBytes int64
	Allowed  map[string]bool
}

func (s LocalAttachmentStore) Save(ctx context.Context, name string, r io.Reader) (string, string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", "", 0, err
	}
	base := filepath.Base(name)
	if base != name || strings.Contains(name, "..") {
		return "", "", 0, fmt.Errorf("invalid attachment name")
	}
	if !s.Allowed[filepath.Ext(base)] {
		return "", "", 0, fmt.Errorf("attachment type not allowed")
	}
	if err := os.MkdirAll(s.Root, 0o750); err != nil {
		return "", "", 0, err
	}
	f, err := os.CreateTemp(s.Root, "upload-*")
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, s.MaxBytes+1))
	if err != nil {
		return "", "", 0, err
	}
	if n > s.MaxBytes {
		return "", "", 0, fmt.Errorf("attachment too large")
	}
	path := f.Name()
	sum := hex.EncodeToString(h.Sum(nil))
	return path, sum, n, nil
}
