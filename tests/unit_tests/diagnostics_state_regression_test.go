package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckIfResultsExistReturnsFalseWhenCurrentReportWasDeleted(t *testing.T) {
	root := t.TempDir()
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(results, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := processing.NewApp(reporting.NoOpReporter{})
	app.Startup(context.Background())
	if err := app.StartExecution(models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{root},
		ResultsDir:      results,
		CacheDir:        cache,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}); err != nil {
		t.Fatalf("run duplicate scan: %v", err)
	}
	result := app.GetResults()
	if result == nil || result.ReportPath == "" {
		t.Fatalf("missing duplicate report path: %#v", result)
	}
	if err := os.Remove(result.ReportPath); err != nil {
		t.Fatalf("remove report: %v", err)
	}
	if app.CheckIfResultsExist() {
		t.Fatal("CheckIfResultsExist returned true for a deleted report")
	}
}
