package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"sync"
	"testing"
)

func TestRemoveUnmatchedSizeFiles(t *testing.T) {
	files := &sync.Map{}
	files.Store("match", models.FileHash{FilePath: "match", FileSize: 10})
	files.Store("skip", models.FileHash{FilePath: "skip", FileSize: 20})
	files.Store("also-match", models.FileHash{FilePath: "also-match", FileSize: 10})

	candidates, skipped := services.RemoveUnmatchedSizeFiles(files, map[int64]struct{}{10: {}})
	if candidates != 2 || skipped != 1 {
		t.Fatalf("expected 2 candidates and 1 skipped file, got %d and %d", candidates, skipped)
	}

	if _, ok := files.Load("skip"); ok {
		t.Fatal("expected disallowed-size file to be removed")
	}
	for _, path := range []string{"match", "also-match"} {
		if _, ok := files.Load(path); !ok {
			t.Fatalf("expected allowed-size file %q to remain", path)
		}
	}
}

func TestRemoveUnmatchedSizeFilesWithEmptySet(t *testing.T) {
	files := &sync.Map{}
	files.Store("file", models.FileHash{FilePath: "file", FileSize: 0})

	candidates, skipped := services.RemoveUnmatchedSizeFiles(files, nil)
	if candidates != 0 || skipped != 1 {
		t.Fatalf("expected 0 candidates and 1 skipped file, got %d and %d", candidates, skipped)
	}
}
