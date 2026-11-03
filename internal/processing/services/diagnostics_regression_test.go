package services

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCreateHashesStopsAfterFatalDisposition(t *testing.T) {
	files := &sync.Map{}
	for i := 0; i < 3; i++ {
		path := filepath.Join(t.TempDir(), "directory")
		files.Store(path, models.FileHash{FilePath: path})
	}
	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Hashing")
	tracker.Start()
	memory := NewMemoryManager(&models.ExecutionParams{}, 1, 1)
	var calls atomic.Int32
	sink := NewDiagnosticSink()
	defer sink.Close()
	defer tracker.Wait()

	err := CreateHashes(context.Background(), files, 1, tracker, memory, &map[string]models.FileHash{}, sink, func(DiagnosticEvent) ErrorDisposition {
		calls.Add(1)
		return ErrorFatal
	}, models.DiagnosticScopeDuplicate)
	if err == nil {
		t.Fatal("expected fatal hash error")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("fatal handler called %d times, want 1", got)
	}
	remaining := 0
	files.Range(func(_, _ any) bool {
		remaining++
		return true
	})
	if remaining == 0 {
		t.Fatal("fatal hash disposition was applied only after all files were processed")
	}
}

func TestDiscoverWithSinkUsesWarningSeverity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	sink := NewDiagnosticSink()
	defer sink.Close()

	var severity DiagnosticSeverity
	_, err := NewFileDiscovery().DiscoverWithSink(
		context.Background(),
		[]string{root},
		reporting.NoOpReporter{},
		sink,
		func(event DiagnosticEvent) ErrorDisposition {
			severity = event.Severity
			return ErrorDiagnostic
		},
		models.DiagnosticScopeDuplicate,
	)
	if err != nil {
		t.Fatalf("discover missing root: %v", err)
	}
	if severity != DiagnosticSeverityWarning {
		t.Fatalf("severity = %q, want %q for a recoverable walk failure", severity, DiagnosticSeverityWarning)
	}
}

func TestCoverageCSVUsesLowercaseScope(t *testing.T) {
	var output strings.Builder
	err := WriteCoverageCSV(&output, models.MissingContentResult{}, models.ScanDiagnostics{
		Diagnostics: []models.Diagnostic{{
			Path:   "/source/private",
			Reason: "permission denied",
			Phase:  models.DiagnosticPhaseWalking,
			Scope:  models.DiagnosticScopeSource,
		}},
	})
	if err != nil {
		t.Fatalf("write coverage CSV: %v", err)
	}
	if !strings.Contains(output.String(), "Skipped,source,") {
		t.Fatalf("coverage CSV does not use lowercase source scope: %q", output.String())
	}
}

func TestSaveDuplicateReportIncludesDiagnosticOnlyRows(t *testing.T) {
	path, err := SaveDuplicateReport(&sync.Map{}, models.ScanDiagnostics{
		Diagnostics: []models.Diagnostic{{
			Path:   "/data/private",
			Reason: "permission denied",
			Phase:  models.DiagnosticPhaseWalking,
			Scope:  models.DiagnosticScopeDuplicate,
		}},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("save diagnostic-only report: %v", err)
	}
	if path == "" {
		t.Fatal("expected diagnostic-only report path")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Skipped") || !strings.Contains(text, "permission denied (phase: walking)") {
		t.Fatalf("report does not contain diagnostic row: %q", text)
	}
}

func TestSaveDuplicateReportDoesNotCreateEmptyReport(t *testing.T) {
	path, err := SaveDuplicateReport(&sync.Map{}, models.ScanDiagnostics{}, t.TempDir())
	if err != nil {
		t.Fatalf("save empty report: %v", err)
	}
	if path != "" {
		t.Fatalf("empty report path = %q, want empty", path)
	}
}

func TestSaveDuplicateReportUsesUniquePathsForRapidRuns(t *testing.T) {
	directory := t.TempDir()
	diagnostics := models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{
		Path:   "/data/private",
		Reason: "permission denied",
		Phase:  models.DiagnosticPhaseWalking,
		Scope:  models.DiagnosticScopeDuplicate,
	}}}
	first, err := SaveDuplicateReport(&sync.Map{}, diagnostics, directory)
	if err != nil {
		t.Fatalf("save first report: %v", err)
	}
	second, err := SaveDuplicateReport(&sync.Map{}, diagnostics, directory)
	if err != nil {
		t.Fatalf("save second report: %v", err)
	}
	if first == second {
		t.Fatalf("rapid reports reused path %q", first)
	}
}
