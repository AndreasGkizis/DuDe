package unit_tests

import (
	"sync"
	"testing"

	"DuDe/internal/models"
	"DuDe/internal/visuals"
)

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

	tracker := visuals.NewProgressTracker("Test Progress")
	tracker.Start(10)

	// ACT
	process.FindDuplicatesInMap(fileHashes, tracker)

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
