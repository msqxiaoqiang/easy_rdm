package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBasePath_WithConfig(t *testing.T) {
	result := ResolveBasePath("/custom/path")
	if result != "/custom/path" {
		t.Fatalf("expected /custom/path, got %s", result)
	}
}

func TestResolveBasePath_Empty(t *testing.T) {
	result := ResolveBasePath("")
	if result == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestExeRelative(t *testing.T) {
	result := ExeRelative("../../config")
	if result == "" {
		t.Fatal("expected non-empty path")
	}
	// 应该是绝对路径
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got %s", result)
	}
}

func TestWritePortToFile(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "app.port")

	err := WritePortToFile(portFile, 8899)
	if err != nil {
		t.Fatalf("WritePortToFile error: %v", err)
	}

	data, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatalf("read port file error: %v", err)
	}
	if string(data) != "8899" {
		t.Fatalf("expected 8899, got %s", string(data))
	}
}

func TestRemoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(f, []byte("test"), 0644)

	RemoveFile(f)

	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatal("file should be removed")
	}
}

func TestRemoveFile_NotExist(t *testing.T) {
	// 不应 panic
	RemoveFile("/nonexistent/path/file.txt")
}
