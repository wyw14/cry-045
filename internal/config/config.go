package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	LocalFileRoot    string
	Environment      string
	UploadLimitBytes int64
	ReviewGraceHours int
	AllowedOrigins   []string
}

func Load() Config {
	c := Config{HTTPAddr: ":8080", DatabaseURL: "", LocalFileRoot: "./var/attachments", Environment: "local", UploadLimitBytes: 10 << 20, ReviewGraceHours: 48, AllowedOrigins: []string{"http://localhost"}}
	if value := os.Getenv("HTTP_ADDR"); value != "" {
		c.HTTPAddr = value
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		c.DatabaseURL = value
	}
	if value := os.Getenv("LOCAL_FILE_ROOT"); value != "" {
		c.LocalFileRoot = value
	}
	if value := os.Getenv("APP_ENV"); value != "" {
		c.Environment = strings.ToLower(value)
	}
	if value := os.Getenv("UPLOAD_LIMIT_BYTES"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			c.UploadLimitBytes = parsed
		}
	}
	if value := os.Getenv("REVIEW_GRACE_HOURS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			c.ReviewGraceHours = parsed
		}
	}
	if value := os.Getenv("ALLOWED_ORIGINS"); value != "" {
		c.AllowedOrigins = splitNonEmpty(value)
	}
	return c
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("HTTP_ADDR is required")
	}
	if c.UploadLimitBytes < 1024 {
		return fmt.Errorf("upload limit is too small")
	}
	if c.ReviewGraceHours < 1 || c.ReviewGraceHours > 720 {
		return fmt.Errorf("review grace hours out of range")
	}
	return nil
}

func (c Config) IsProduction() bool { return c.Environment == "production" }

func splitNonEmpty(value string) []string {
	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
