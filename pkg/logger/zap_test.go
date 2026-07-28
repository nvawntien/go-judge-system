package logger

import (
	"os"
	"path/filepath"
	"testing"

	"go-judge-system/pkg/config"
)

func TestNewLoggerSkipsFileSinkWhenFilenameIsEmpty(t *testing.T) {
	t.Parallel()

	logger := NewLogger(config.LoggerConfig{
		Filename: "",
		MaxSize:  1,
	}, "release")

	logger.Info("stdout only logger smoke")
	if err := logger.Sync(); err != nil {
		t.Logf("logger sync returned non-fatal error: %v", err)
	}

}

func TestNewLoggerCreatesFileSinkWhenFilenameIsSet(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "app.log")
	logger := NewLogger(config.LoggerConfig{
		Filename: filename,
		MaxSize:  1,
	}, "release")

	logger.Info("file logger smoke")
	if err := logger.Sync(); err != nil {
		t.Logf("logger sync returned non-fatal error: %v", err)
	}

	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("stat log file: %v", err)
	}
}
