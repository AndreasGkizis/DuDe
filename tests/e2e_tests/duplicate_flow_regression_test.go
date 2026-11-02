package e2e_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDuplicateFlowPublishesGroupsAfterSuccessfulExecution(t *testing.T) {
	app := setupTestApp(t)
	root, cleanup := createTestFilesByteArray(t, map[string][]byte{
		"one.txt":    []byte("same content"),
		"two.txt":    []byte("same content"),
		"unique.txt": []byte("different content"),
	})
	defer func() {
		cleanup()
		deleteTestFolder(t)
	}()

	resultsDir := filepath.Join(root, "results")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := app.StartExecution(models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{root},
		ResultsDir:      resultsDir,
		CacheDir:        cacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	})
	if err != nil {
		t.Fatalf("duplicate execution failed: %v", err)
	}

	groups := app.GetResults()
	if len(groups) != 1 {
		t.Fatalf("expected one duplicate group, got %d", len(groups))
	}
	if len(groups[0].DuplicatesFound) != 1 {
		t.Fatalf("expected one duplicate in the group, got %d", len(groups[0].DuplicatesFound))
	}
}

func TestCancelledDuplicateFlowDoesNotPublishStaleResults(t *testing.T) {
	reporter := &cancelOnProgressReporter{started: make(chan struct{})}
	app := processing.NewApp(reporter)
	app.Startup(context.Background())

	firstRoot, firstCleanup := createTestFilesByteArray(t, map[string][]byte{
		"one.txt": []byte("same content"),
		"two.txt": []byte("same content"),
	})
	defer func() {
		firstCleanup()
		deleteTestFolder(t)
	}()

	firstResults := filepath.Join(firstRoot, "results")
	firstCache := filepath.Join(firstRoot, "cache")
	if err := os.MkdirAll(firstResults, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(firstCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := app.StartExecution(models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{firstRoot},
		ResultsDir:      firstResults,
		CacheDir:        firstCache,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}); err != nil {
		t.Fatalf("initial duplicate execution failed: %v", err)
	}
	if got := len(app.GetResults()); got != 1 {
		t.Fatalf("expected initial execution to publish one group, got %d", got)
	}

	root, cleanup := createTestFilesByteArray(t, duplicateFiles(200))
	defer func() {
		cleanup()
		deleteTestFolder(t)
	}()
	resultsDir := filepath.Join(root, "results")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	reporter.started = make(chan struct{})
	reporter.once = sync.Once{}
	reporter.cancel = app.CancelExecution
	done := make(chan error, 1)
	go func() {
		done <- app.StartExecution(models.ExecutionParams{
			Mode:            models.ExecutionModeDuplicates,
			Directories:     []string{root},
			ResultsDir:      resultsDir,
			CacheDir:        cacheDir,
			WorkerCount:     1,
			BufferSizeBytes: 1024,
		})
	}()

	select {
	case <-reporter.started:
	case <-time.After(5 * time.Second):
		t.Fatal("duplicate execution did not start")
	}
	reporter.cancel()

	if err := <-done; err != nil {
		t.Fatalf("cancelled duplicate execution failed: %v", err)
	}
	if groups := app.GetResults(); groups != nil {
		t.Fatalf("expected cancelled execution to publish no groups, got %d", len(groups))
	}
}

func duplicateFiles(count int) map[string][]byte {
	files := make(map[string][]byte, count)
	for i := 0; i < count; i++ {
		files[filepath.Join("files", fmt.Sprintf("file-%03d.txt", i))] = []byte("identical candidate content")
	}
	return files
}

type cancelOnProgressReporter struct {
	started chan struct{}
	cancel  func()
	once    sync.Once
}

func (r *cancelOnProgressReporter) LogProgress(_ context.Context, title string, _ float64) {
	if title == "Reading" {
		r.once.Do(func() { close(r.started) })
	}
}

func (*cancelOnProgressReporter) LogDetailedStatus(context.Context, string)   {}
func (*cancelOnProgressReporter) LogFilesCount(context.Context, int64, int64) {}
func (*cancelOnProgressReporter) FinishExecution(context.Context)             {}
func (*cancelOnProgressReporter) ReportError(context.Context, string)         {}

var _ reporting.Reporter = (*cancelOnProgressReporter)(nil)
