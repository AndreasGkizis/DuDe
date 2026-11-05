package unit_tests

import (
	"os"
	"path/filepath"
	"testing"

	commonfs "DuDe/internal/common/fs"
)

func TestCanWriteDoesNotModifyExistingProbeNamedFile(t *testing.T) {
	directory := t.TempDir()
	existingPath := filepath.Join(directory, ".tmp_write_test")
	existingContent := []byte("keep this content")
	if err := os.WriteFile(existingPath, existingContent, 0o600); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	if !(commonfs.OS{}).CanWrite(directory) {
		t.Fatal("expected temporary directory to be writable")
	}

	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("read existing file after writable check: %v", err)
	}
	if string(content) != string(existingContent) {
		t.Fatalf("expected existing file content to remain unchanged, got %q", content)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read directory after writable check: %v", err)
	}
	for _, entry := range entries {
		if matched, _ := filepath.Match(".dude-write-test-*", entry.Name()); matched {
			t.Fatalf("writable check left probe file %q behind", entry.Name())
		}
	}
}
