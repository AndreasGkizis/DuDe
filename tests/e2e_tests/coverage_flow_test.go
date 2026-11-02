package e2e_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestCoverageFlowSupportsParanoidMode(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	writeCoverageFile(t, filepath.Join(source, "source.txt"), "identical bytes")
	writeCoverageFile(t, filepath.Join(target, "renamed.txt"), "identical bytes")

	app := processing.NewApp(reporting.NoOpReporter{})
	app.Startup(context.Background())
	params := coverageParams(source, target, results, cache)
	params.ParanoidMode = true
	if err := app.StartExecution(params); err != nil {
		t.Fatalf("paranoid coverage execution failed: %v", err)
	}
	if result := app.GetMissingContentResult(); result == nil || len(result.MissingFiles) != 0 {
		t.Fatalf("expected paranoid byte-identical file to be covered, got %#v", result)
	}
}

func TestCoverageFlowRetainsSkippedPathDiagnostics(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	writeCoverageFile(t, filepath.Join(source, "covered.txt"), "content")
	writeCoverageFile(t, filepath.Join(target, "renamed.txt"), "content")
	skippedSource := filepath.Join(source, "private")
	skippedTarget := filepath.Join(target, "private")
	writeCoverageFile(t, filepath.Join(skippedSource, "source-only.txt"), "source-only")
	writeCoverageFile(t, filepath.Join(skippedTarget, "target-only.txt"), "target-only")
	if err := os.Chmod(skippedSource, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(skippedTarget, 0); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(skippedSource, 0o755)
		_ = os.Chmod(skippedTarget, 0o755)
	}()

	app := processing.NewApp(reporting.NoOpReporter{})
	app.Startup(context.Background())
	if err := app.StartExecution(coverageParams(source, target, results, cache)); err != nil {
		t.Fatalf("coverage execution failed: %v", err)
	}
	diagnostics := app.GetScanDiagnostics()
	if diagnostics == nil || (len(diagnostics.SkippedSourcePaths) == 0 && len(diagnostics.SkippedTargetPaths) == 0) {
		t.Skip("filesystem permissions are readable by the test user")
	}
	if result := app.GetMissingContentResult(); result == nil || result.TotalSourceFiles != 1 || len(result.MissingFiles) != 0 {
		t.Fatalf("unexpected coverage result with skipped paths: %#v", result)
	}
}

func TestCoverageFlowSuppressesResultAfterInconclusiveReportFailure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	writeCoverageFile(t, filepath.Join(source, "source.txt"), "content")
	writeCoverageFile(t, filepath.Join(target, "target.txt"), "content")

	reporter := &removeResultsReporter{resultsDir: results}
	app := processing.NewApp(reporter)
	app.Startup(context.Background())
	if err := app.StartExecution(coverageParams(source, target, results, cache)); err == nil {
		t.Fatal("expected report failure to make coverage inconclusive")
	}
	if result := app.GetMissingContentResult(); result != nil {
		t.Fatalf("expected no published result after inconclusive execution, got %#v", result)
	}
	entries, err := os.ReadDir(results)
	if err == nil && len(entries) != 0 {
		t.Fatalf("expected no coverage report after inconclusive execution, found %v", entries)
	}
}

func TestCoverageFlowCancellationDoesNotPublishStaleResults(t *testing.T) {
	reporter := &cancelOnProgressReporter{started: make(chan struct{})}
	app := processing.NewApp(reporter)
	app.Startup(context.Background())

	firstRoot := t.TempDir()
	firstSource := filepath.Join(firstRoot, "source")
	firstTarget := filepath.Join(firstRoot, "target")
	firstResults := filepath.Join(firstRoot, "results")
	firstCache := filepath.Join(firstRoot, "cache")
	createCoverageDir(t, firstResults)
	createCoverageDir(t, firstCache)
	writeCoverageFile(t, filepath.Join(firstSource, "source.txt"), "content")
	writeCoverageFile(t, filepath.Join(firstTarget, "target.txt"), "content")
	if err := app.StartExecution(coverageParams(firstSource, firstTarget, firstResults, firstCache)); err != nil {
		t.Fatalf("initial coverage execution failed: %v", err)
	}
	if app.GetMissingContentResult() == nil {
		t.Fatal("expected initial coverage result")
	}

	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	for i := 0; i < 300; i++ {
		writeCoverageFile(t, filepath.Join(source, "files", fmt.Sprintf("file-%03d.txt", i)), "content")
	}
	writeCoverageFile(t, filepath.Join(target, "target.txt"), "content")

	reporter.started = make(chan struct{})
	reporter.once = sync.Once{}
	reporter.cancel = app.CancelExecution
	done := make(chan error, 1)
	go func() { done <- app.StartExecution(coverageParams(source, target, results, cache)) }()
	select {
	case <-reporter.started:
	case <-time.After(5 * time.Second):
		t.Fatal("coverage execution did not start")
	}
	reporter.cancel()
	if err := <-done; err != nil {
		t.Fatalf("cancelled coverage execution failed: %v", err)
	}
	if result := app.GetMissingContentResult(); result != nil {
		t.Fatalf("expected cancelled execution to publish no result, got %#v", result)
	}
	if entries, err := os.ReadDir(results); err == nil && len(entries) != 0 {
		t.Fatalf("expected cancelled execution to produce no report, found %v", entries)
	}
}

func TestFullResetDuringCoverageClearsResultState(t *testing.T) {
	reporter := &cancelOnProgressReporter{started: make(chan struct{})}
	app := processing.NewApp(reporter)
	app.Startup(context.Background())
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	results := filepath.Join(root, "results")
	cache := filepath.Join(root, "cache")
	createCoverageDir(t, results)
	createCoverageDir(t, cache)
	for i := 0; i < 300; i++ {
		writeCoverageFile(t, filepath.Join(source, "files", fmt.Sprintf("file-%03d.txt", i)), "content")
	}
	writeCoverageFile(t, filepath.Join(target, "target.txt"), "content")

	reporter.cancel = nil
	done := make(chan error, 1)
	go func() { done <- app.StartExecution(coverageParams(source, target, results, cache)) }()
	select {
	case <-reporter.started:
	case <-time.After(5 * time.Second):
		t.Fatal("coverage execution did not start")
	}
	resetDone := make(chan error, 1)
	go func() { resetDone <- app.FullReset() }()
	if err := <-done; err != nil {
		t.Fatalf("reset-cancelled coverage execution failed: %v", err)
	}
	if err := <-resetDone; err != nil {
		t.Fatalf("full reset failed: %v", err)
	}
	if result := app.GetMissingContentResult(); result != nil {
		t.Fatalf("expected full reset to clear coverage result, got %#v", result)
	}
	if args := app.Args; args.Mode != 0 || args.FindMissingSourceDir != "" || args.FindMissingTargetDir != "" {
		t.Fatalf("expected full reset to clear execution args, got %#v", args)
	}
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

type removeResultsReporter struct {
	reporting.NoOpReporter
	resultsDir string
	once       sync.Once
}

func (r *removeResultsReporter) LogProgress(_ context.Context, title string, _ float64) {
	if title == "Reporting" {
		r.once.Do(func() { _ = os.RemoveAll(r.resultsDir) })
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
