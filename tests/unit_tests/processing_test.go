package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

func TestMain(m *testing.M) {
	// Setup - runs ONCE before ALL tests in this package
	log.Initialize(false)

	// Run all tests
	exitCode := m.Run()
	// Exit with the test result code
	os.Exit(exitCode)
}

func TestFindDuplicatesInMap(t *testing.T) {
	// ARRANGE
	fileHashes := &sync.Map{}

	fileHashes.Store("file1", models.FileHash{
		FileName: "file1",
		FilePath: "/path/to/file1",
		Hash:     "abc123",
	})
	fileHashes.Store("file2", models.FileHash{
		FileName: "file2",
		FilePath: "/path/to/file2",
		Hash:     "def456",
	})
	fileHashes.Store("file3", models.FileHash{
		FileName: "file3",
		FilePath: "/path/to/file3",
		Hash:     "abc123", // Duplicate hash
	})

	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Test Progress")
	tracker.Start()

	// ACT
	processing.FindDuplicatesInMap(context.Background(), fileHashes, tracker)

	// ASSERT
	fileHashes.Range(func(key, value any) bool {
		fileHash := value.(models.FileHash)
		if fileHash.Hash == "abc123" {
			if len(fileHash.DuplicatesFound) != 1 {
				t.Errorf("Expected 1 duplicate for hash abc123, got %d", len(fileHash.DuplicatesFound))
			}
		}
		return true
	})

	tracker.Wait()
}

func TestWalkDirStoresFileMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "sample.txt")
	contents := []byte("sample contents")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	files := &sync.Map{}
	counter := visuals.NewProgressCounter(context.Background(), reporting.NoOpReporter{}, "test", 1)
	counter.Start()
	processing.WalkDir(context.Background(), directory, files, counter)
	counter.WaitForSenders()
	counter.Wg.Wait()

	stored, ok := files.Load(path)
	if !ok {
		t.Fatal("expected walked file to be stored")
	}

	file := stored.(models.FileHash)
	if file.FileName != "sample.txt" {
		t.Errorf("expected filename sample.txt, got %q", file.FileName)
	}
	if file.FileSize != int64(len(contents)) {
		t.Errorf("expected size %d, got %d", len(contents), file.FileSize)
	}
	if file.ModTime == "" {
		t.Error("expected modification time to be stored")
	}
}

func TestFilterHashCandidatesBySize(t *testing.T) {
	tests := []struct {
		name             string
		files            map[string]int64
		wantCandidates   int
		wantSkipped      int
		wantRemainingIDs map[string]bool
	}{
		{
			name:           "all sizes unique",
			files:          map[string]int64{"a": 1, "b": 2, "c": 3},
			wantCandidates: 0,
			wantSkipped:    3,
		},
		{
			name:             "same size files remain candidates",
			files:            map[string]int64{"a": 10, "b": 10, "c": 20},
			wantCandidates:   2,
			wantSkipped:      1,
			wantRemainingIDs: map[string]bool{"a": true, "b": true},
		},
		{
			name:             "multiple repeated size groups",
			files:            map[string]int64{"a": 10, "b": 10, "c": 20, "d": 20, "e": 30},
			wantCandidates:   4,
			wantSkipped:      1,
			wantRemainingIDs: map[string]bool{"a": true, "b": true, "c": true, "d": true},
		},
		{
			name:             "empty files remain candidates",
			files:            map[string]int64{"a": 0, "b": 0, "c": 1},
			wantCandidates:   2,
			wantSkipped:      1,
			wantRemainingIDs: map[string]bool{"a": true, "b": true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			files := &sync.Map{}
			for path, size := range test.files {
				files.Store(path, models.FileHash{FilePath: path, FileSize: size})
			}

			candidates, skipped := processing.FilterHashCandidatesBySize(files)
			if candidates != test.wantCandidates {
				t.Errorf("expected %d candidates, got %d", test.wantCandidates, candidates)
			}
			if skipped != test.wantSkipped {
				t.Errorf("expected %d skipped files, got %d", test.wantSkipped, skipped)
			}

			remaining := make(map[string]bool)
			files.Range(func(key, _ any) bool {
				remaining[key.(string)] = true
				return true
			})
			if len(remaining) != len(test.wantRemainingIDs) {
				t.Fatalf("expected %d remaining files, got %d", len(test.wantRemainingIDs), len(remaining))
			}
			for path := range test.wantRemainingIDs {
				if !remaining[path] {
					t.Errorf("expected %q to remain", path)
				}
			}
		})
	}
}

func TestSameSizeDifferentFilesAreHashedButNotDuplicates(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.txt")
	secondPath := filepath.Join(directory, "second.txt")
	if err := os.WriteFile(firstPath, []byte("aaaa"), 0o600); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("bbbb"), 0o600); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	files := &sync.Map{}
	files.Store(firstPath, models.FileHash{FilePath: firstPath, FileSize: 4})
	files.Store(secondPath, models.FileHash{FilePath: secondPath, FileSize: 4})

	candidates, skipped := processing.FilterHashCandidatesBySize(files)
	if candidates != 2 || skipped != 0 {
		t.Fatalf("expected 2 candidates and 0 skipped files, got %d and %d", candidates, skipped)
	}

	runCreateHashes(t, files, make(chan error, 2))
	first := loadFileHash(t, files, firstPath)
	second := loadFileHash(t, files, secondPath)
	if first.Hash == "" || second.Hash == "" {
		t.Fatal("expected both same-sized files to be hashed")
	}
	if first.Hash == second.Hash {
		t.Fatal("expected different contents to produce different hashes")
	}

	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "test")
	processing.FindDuplicatesInMap(context.Background(), files, tracker)
	remaining := 0
	files.Range(func(_, _ any) bool {
		remaining++
		return true
	})
	if remaining != 0 {
		t.Fatalf("expected no duplicate groups, got %d", remaining)
	}
}

func loadFileHash(t *testing.T, files *sync.Map, path string) models.FileHash {
	t.Helper()

	value, ok := files.Load(path)
	if !ok {
		t.Fatalf("expected %q in file map", path)
	}
	return value.(models.FileHash)
}
