package processing

import (
	"sync"
	"testing"

	"DuDe/internal/models"
	"DuDe/internal/visuals"
)

func TestFindDuplicatesBetweenMaps(t *testing.T) {
	// Create mock data for the first map
	first := &sync.Map{}
	first.Store("file1", models.FileHash{
		FileName: "file1",
		FilePath: "/path/to/file1",
		Hash:     "abc123",
	})
	first.Store("file2", models.FileHash{
		FileName: "file2",
		FilePath: "/path/to/file2",
		Hash:     "def456",
	})

	// Create mock data for the second map
	second := &sync.Map{}
	second.Store("file3", models.FileHash{
		FileName: "file3",
		FilePath: "/path/to/file3",
		Hash:     "abc123", // Duplicate hash
	})
	second.Store("file4", models.FileHash{
		FileName: "file4",
		FilePath: "/path/to/file4",
		Hash:     "ghi789",
	})

	// Create a mock progress tracker
	tracker := visuals.NewProgressTracker("Test Progress")
	tracker.Start(10)

	// Call the function
	FindDuplicatesBetweenMaps(first, second, tracker, 8)

	// Verify the results
	first.Range(func(key, value any) bool {
		fileHash := value.(models.FileHash)
		if fileHash.FileName == "file1" {
			if len(fileHash.DuplicatesFound) != 1 {
				t.Errorf("Expected 1 duplicate for file1, got %d", len(fileHash.DuplicatesFound))
			}
			if fileHash.DuplicatesFound[0].FileName != "file3" {
				t.Errorf("Expected duplicate to be file3, got %s", fileHash.DuplicatesFound[0].FileName)
			}
		}
		return true
	})

	tracker.Wait()
}

func TestFindDuplicatesInMap(t *testing.T) {
	// Create mock data for the sync.Map
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

	// Create a mock progress tracker
	tracker := visuals.NewProgressTracker("Test Progress")
	tracker.Start(10)

	// Call the function
	FindDuplicatesInMap(fileHashes, tracker)

	// Verify the results
	fileHashes.Range(func(key, value any) bool {
		fileHash := value.(models.FileHash)
		if fileHash.Hash == "abc123" {
			if len(fileHash.DuplicatesFound) != 1 {
				t.Errorf("Expected 1 duplicate for hash abc123, got %d", len(fileHash.DuplicatesFound))
			}
			if fileHash.DuplicatesFound[0].FileName != "file3" {
				t.Errorf("Expected duplicate to be file3, got %s", fileHash.DuplicatesFound[0].FileName)
			}
		}
		return true
	})

	tracker.Wait()
}
