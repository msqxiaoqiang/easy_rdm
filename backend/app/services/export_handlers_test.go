package services

import (
	"encoding/json"
	"testing"
)

// ========== exportFileExt 测试 ==========

func TestExportFileExt(t *testing.T) {
	tests := []struct {
		format string
		ext    string
	}{
		{"json", ".json"},
		{"csv", ".csv"},
		{"redis_cmd", ".txt"},
		{"", ".json"},
		{"unknown", ".json"},
	}
	for _, tt := range tests {
		ext := exportFileExt(tt.format)
		if ext != tt.ext {
			t.Errorf("format=%q: expected %s, got %s", tt.format, tt.ext, ext)
		}
	}
}

// ========== handleExportKeysFile 测试 ==========

func TestExportKeysFile_InvalidParams(t *testing.T) {
	_, err := handleExportKeysFile([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestExportKeysFile_MissingFilePath(t *testing.T) {
	params, _ := json.Marshal(map[string]interface{}{
		"conn_id": "test",
		"format":  "json",
		"scope":   "all",
	})
	_, err := handleExportKeysFile(params)
	if err == nil {
		t.Fatal("expected error for missing file_path")
	}
}

// ========== handleImportKeysFile 测试 ==========

func TestImportKeysFile_InvalidParams(t *testing.T) {
	_, err := handleImportKeysFile([]byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestImportKeysFile_MissingFilePath(t *testing.T) {
	params, _ := json.Marshal(map[string]interface{}{
		"conn_id": "test",
		"format":  "json",
	})
	_, err := handleImportKeysFile(params)
	if err == nil {
		t.Fatal("expected error for missing file_path")
	}
}

func TestImportKeysFile_FileNotExist(t *testing.T) {
	params, _ := json.Marshal(map[string]interface{}{
		"conn_id":   "test",
		"format":    "json",
		"file_path": "/tmp/nonexistent_easy_rdm_test_12345.json",
	})
	_, err := handleImportKeysFile(params)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
