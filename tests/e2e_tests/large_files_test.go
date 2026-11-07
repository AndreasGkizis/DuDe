package e2e_tests

import (
	"DuDe/internal/models"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLargeFilesWithConstrainedBuffer(t *testing.T) {
	const fileSize = 4 * 1024 * 1024

	content := bytes.Repeat([]byte("DuDe-large-file-test-"), fileSize/len("DuDe-large-file-test-"))
	files := map[string][]byte{
		"large-a.bin": content,
		"large-b.bin": append([]byte(nil), content...),
		"unique.bin":  bytes.Repeat([]byte{'u'}, fileSize),
	}
	root, cleanup := createTestFilesByteArray(t, files)
	defer cleanup()

	resultsDir := filepath.Join(root, "results")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	app := setupTestApp(t)
	err := app.StartExecution(models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{root},
		ResultsDir:      resultsDir,
		CacheDir:        cacheDir,
		WorkerCount:     2,
		BufferSizeBytes: 4096,
	})
	if err != nil {
		t.Fatalf("large-file execution failed: %v", err)
	}

	result := app.GetResults()
	if result == nil || len(result.Groups) != 1 {
		t.Fatalf("expected one large-file duplicate group, got %#v", result)
	}
	if len(result.Groups[0].DuplicatesFound) != 1 {
		t.Fatalf("expected one duplicate in the large-file group, got %d", len(result.Groups[0].DuplicatesFound))
	}
}
