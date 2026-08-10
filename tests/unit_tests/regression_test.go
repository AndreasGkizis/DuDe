package unit_tests

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

func TestCreateHashesReportsReadErrors(t *testing.T) {
	sourceFiles := &sync.Map{}
	directory := t.TempDir()
	sourceFiles.Store(directory, models.FileHash{FilePath: directory})

	errChan := make(chan error, 1)
	runCreateHashes(t, sourceFiles, errChan)

	select {
	case <-errChan:
	default:
		t.Fatal("expected hashing a directory to report its read error")
	}
}

func TestProgressTrackerNeverReportsNonFinitePercentage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reporter := capturingReporter{progress: make(chan progressEvent, 2)}
	tracker := visuals.NewProgressTracker(ctx, reporter, "test")
	tracker.Start()
	<-reporter.progress
	tracker.Increment()

	var update progressEvent
	select {
	case update = <-reporter.progress:
	case <-time.After(time.Second):
		cancel()
		tracker.Wait()
		t.Fatal("timed out waiting for progress update")
	}

	cancel()
	tracker.Wait()

	if math.IsInf(update.percent, 0) || math.IsNaN(update.percent) {
		t.Fatalf("expected a finite progress percentage, got %v", update.percent)
	}
}

func TestProgressTrackerReportsPhaseImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reporter := capturingReporter{progress: make(chan progressEvent, 1)}
	tracker := visuals.NewProgressTracker(ctx, reporter, "Finding")

	tracker.Start()
	update := <-reporter.progress
	cancel()
	tracker.Wait()

	if update.title != "Finding" {
		t.Fatalf("expected Finding phase, got %q", update.title)
	}
	if update.percent != 0 {
		t.Fatalf("expected phase to start at 0%%, got %v", update.percent)
	}
}

func runCreateHashes(t *testing.T, sourceFiles *sync.Map, errChan chan error) {
	t.Helper()

	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "test")
	memoryManager := processing.NewMemoryManager(&models.ExecutionParams{}, 1, 1)
	memory := make(map[string]models.FileHash)
	failedCount := 0

	err := processing.CreateHashes(
		context.Background(),
		sourceFiles,
		1,
		tracker,
		memoryManager,
		&memory,
		&failedCount,
		errChan,
	)
	if err != nil {
		t.Fatalf("create hashes: %v", err)
	}
}

type capturingReporter struct {
	progress chan progressEvent
}

type progressEvent struct {
	title   string
	percent float64
}

func (reporter capturingReporter) LogProgress(_ context.Context, title string, percent float64) {
	reporter.progress <- progressEvent{title: title, percent: percent}
}

func (capturingReporter) LogDetailedStatus(context.Context, string)   {}
func (capturingReporter) LogFilesCount(context.Context, int64, int64) {}
func (capturingReporter) FinishExecution(context.Context)             {}
