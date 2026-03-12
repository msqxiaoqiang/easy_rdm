package services

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStorage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	InitStorage(dir)
	return dir
}

func TestWriteJSON_ReadJSON_RoundTrip(t *testing.T) {
	setupTestStorage(t)

	type Item struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := []Item{{Name: "Alice", Age: 30}, {Name: "Bob", Age: 25}}
	if err := WriteJSON("test.json", input); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var output []Item
	if err := ReadJSON("test.json", &output); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}

	if len(output) != 2 || output[0].Name != "Alice" || output[1].Age != 25 {
		t.Fatalf("unexpected output: %+v", output)
	}
}

func TestReadJSON_FileNotExist(t *testing.T) {
	setupTestStorage(t)

	var data map[string]interface{}
	err := ReadJSON("nonexistent.json", &data)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteJSON_AutoCreateDir(t *testing.T) {
	setupTestStorage(t)

	err := WriteJSON("subdir/nested/data.json", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("WriteJSON should auto-create parent dirs: %v", err)
	}

	var result map[string]string
	if err := ReadJSON("subdir/nested/data.json", &result); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}
	if result["key"] != "value" {
		t.Fatalf("expected 'value', got %s", result["key"])
	}
}

func TestInitStorage_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	InitStorage(dir)

	expectedDirs := []string{
		"exports",
	}
	for _, d := range expectedDirs {
		p := filepath.Join(dir, d)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("directory %s should exist: %v", d, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s should be a directory", d)
		}
	}
}

func TestInitStorage_CreatesDefaultFiles(t *testing.T) {
	dir := t.TempDir()
	InitStorage(dir)

	expectedFiles := []string{
		"connections.json",
		"settings.json",
		"session.json",
		"groups.json",
	}
	for _, f := range expectedFiles {
		p := filepath.Join(dir, f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("file %s should exist", f)
		}
	}
}

func TestWriteJSON_Overwrite(t *testing.T) {
	setupTestStorage(t)

	WriteJSON("overwrite.json", map[string]int{"v": 1})
	WriteJSON("overwrite.json", map[string]int{"v": 2})

	var result map[string]int
	ReadJSON("overwrite.json", &result)
	if result["v"] != 2 {
		t.Fatalf("expected 2, got %d", result["v"])
	}
}
