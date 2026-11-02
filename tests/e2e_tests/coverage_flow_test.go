package e2e_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCoverageFlowPublishesCompleteResultAndEmptyReport(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	writeCoverageFile(t, filepath.Join(source, "original.txt"), "same")
	writeCoverageFile(t, filepath.Join(target, "reorganized", "renamed.txt"), "same")
	writeCoverageFile(t, filepath.Join(target, "extra.txt"), "target only")

	app := processing.NewApp(reporting.NoOpReporter{})
	app.Startup(context.Background())
	if err := app.StartExecution(coverageParams(source, target, results, cache)); err != nil {
		t.Fatalf("coverage execution failed: %v", err)
	}

	result := app.GetMissingContentResult()
	if result == nil || result.TotalSourceFiles != 1 || len(result.MissingFiles) != 0 {
		t.Fatalf("unexpected complete coverage result: %#v", result)
	}
	if result.ReportPath == "" {
		t.Fatal("expected coverage report path")
	}
	assertCoverageCSV(t, result.ReportPath, [][]string{{"Type", "Scope", "Filename", "Full Path", "Size (bytes)", "Reason"}})
}

func TestCoverageFlowPublishesEveryMissingSourcePathInOrder(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	writeCoverageFile(t, filepath.Join(source, "z.txt"), "missing z")
	writeCoverageFile(t, filepath.Join(source, "a.txt"), "missing a")
	writeCoverageFile(t, filepath.Join(source, "covered.txt"), "covered")
	writeCoverageFile(t, filepath.Join(target, "elsewhere.bin"), "covered")

	app := processing.NewApp(reporting.NoOpReporter{})
	app.Startup(context.Background())
	if err := app.StartExecution(coverageParams(source, target, results, cache)); err != nil {
		t.Fatalf("coverage execution failed: %v", err)
	}

	result := app.GetMissingContentResult()
	if result == nil || result.TotalSourceFiles != 3 || len(result.MissingFiles) != 2 {
		t.Fatalf("unexpected partial coverage result: %#v", result)
	}
	want := []models.MissingSourceFile{
		{Filename: "a.txt", FullPath: filepath.Join(source, "a.txt"), Size: int64(len("missing a"))},
		{Filename: "z.txt", FullPath: filepath.Join(source, "z.txt"), Size: int64(len("missing z"))},
	}
	if !reflect.DeepEqual(result.MissingFiles, want) {
		t.Fatalf("missing files = %#v, want %#v", result.MissingFiles, want)
	}
}

func coverageParams(source, target, results, cache string) models.ExecutionParams {
	return models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingSourceDir: source,
		FindMissingTargetDir: target,
		ResultsDir:           results,
		CacheDir:             cache,
		WorkerCount:          1,
		BufferSizeBytes:      1024,
	}
}

func createCoverageDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeCoverageFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCoverageCSV(t *testing.T, path string, want [][]string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = services.GetDelimiterForOS()
	got, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > 0 && len(got[0]) > 0 {
		got[0][0] = strings.TrimPrefix(got[0][0], "\ufeff")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coverage CSV = %#v, want %#v", got, want)
	}
}
