package unit_tests

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"testing"
	"time"

	database "DuDe/internal/db"
	"DuDe/internal/models"
	"DuDe/internal/models/db_models"
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

func TestUpsertReturnsUnexpectedUpdateErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	_, err = db.Exec(`CREATE TABLE file_hashes (
		path TEXT UNIQUE,
		hash TEXT,
		size INTEGER,
		modified_time TEXT,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatalf("create malformed schema: %v", err)
	}

	repository := database.NewFileHashRepository(db)
	err = repository.Upsert(&db_models.FileHash{FilePath: "/tmp/file.txt"})
	if err == nil {
		t.Fatal("expected upsert to propagate the lookup error")
	}
}

func TestProgressTrackerNeverReportsNonFinitePercentage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reporter := capturingReporter{progress: make(chan float64, 1)}
	tracker := visuals.NewProgressTracker(ctx, reporter, "test")
	tracker.Start()
	tracker.Increment()

	var percentage float64
	select {
	case percentage = <-reporter.progress:
	case <-time.After(time.Second):
		cancel()
		tracker.Wait()
		t.Fatal("timed out waiting for progress update")
	}

	cancel()
	tracker.Wait()

	if math.IsInf(percentage, 0) || math.IsNaN(percentage) {
		t.Fatalf("expected a finite progress percentage, got %v", percentage)
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
	progress chan float64
}

func (reporter capturingReporter) LogProgress(_ context.Context, _ string, percent float64) {
	reporter.progress <- percent
}

func (capturingReporter) LogDetailedStatus(context.Context, string)   {}
func (capturingReporter) LogFilesCount(context.Context, int64, int64) {}
func (capturingReporter) FinishExecution(context.Context)             {}
