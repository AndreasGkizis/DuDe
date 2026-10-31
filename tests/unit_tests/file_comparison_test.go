package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesMatchAcceptsSameHashWithoutParanoidComparison(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	metadata := models.FileHash{FilePath: path, FileSize: 7, Hash: "same-hash"}
	matched, err := services.FilesMatch(context.Background(), file, metadata, metadata, false)
	if err != nil {
		t.Fatalf("files match: %v", err)
	}
	if !matched {
		t.Fatal("expected equal size and hash to match")
	}
}

func TestFilesMatchRejectsMissingHashesWithoutParanoidComparison(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	metadata := models.FileHash{FilePath: path, FileSize: 7}
	matched, err := services.FilesMatch(context.Background(), file, metadata, metadata, false)
	if err != nil {
		t.Fatalf("files match: %v", err)
	}
	if matched {
		t.Fatal("expected missing hashes not to match in normal mode")
	}
}

func TestFilesMatchRejectsDifferentHashWithoutParanoidComparison(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	primary := models.FileHash{FilePath: path, FileSize: 7, Hash: "first-hash"}
	candidate := models.FileHash{FilePath: path, FileSize: 7, Hash: "second-hash"}
	matched, err := services.FilesMatch(context.Background(), file, primary, candidate, false)
	if err != nil {
		t.Fatalf("files match: %v", err)
	}
	if matched {
		t.Fatal("expected different hashes not to match")
	}
}

func TestFilesMatchRejectsInjectedCollisionWithParanoidComparison(t *testing.T) {
	root := t.TempDir()
	primaryPath := filepath.Join(root, "primary.txt")
	candidatePath := filepath.Join(root, "candidate.txt")
	if err := os.WriteFile(primaryPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, []byte("modified"), 0o600); err != nil {
		t.Fatal(err)
	}
	primaryFile, err := os.Open(primaryPath)
	if err != nil {
		t.Fatal(err)
	}
	defer primaryFile.Close()

	primary := models.FileHash{FilePath: primaryPath, FileSize: 8, Hash: "collision-hash"}
	candidate := models.FileHash{FilePath: candidatePath, FileSize: 8, Hash: "collision-hash"}
	matched, err := services.FilesMatch(context.Background(), primaryFile, primary, candidate, true)
	if err != nil {
		t.Fatalf("files match: %v", err)
	}
	if matched {
		t.Fatal("expected Paranoid Mode to reject unequal files")
	}
}
