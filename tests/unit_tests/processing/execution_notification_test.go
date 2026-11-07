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

func TestExecutionPublishesStateBeforeTerminalNotification(t *testing.T) {
	for _, mode := range []models.ExecutionMode{models.ExecutionModeDuplicates, models.ExecutionModeFindMissingContent} {
		modeName := "duplicates"
		if mode == models.ExecutionModeFindMissingContent {
			modeName = "coverage"
		}
		for _, scenario := range []struct {
			name       string
			files      map[string]string
			cancel     bool
			failReport bool
		}{
			{name: "empty"},
			{name: "unique sizes", files: map[string]string{"a": "a", "b": "bb"}},
			{name: "same size different content", files: map[string]string{"a": "aa", "b": "bb"}},
			{name: "duplicates", files: map[string]string{"a": "same", "b": "same"}},
			{name: "report failure", files: map[string]string{"a": "same", "b": "same"}, failReport: true},
			{name: "cancelled", files: map[string]string{"a": "same", "b": "same"}, cancel: true},
		} {
			t.Run(modeName+"/"+scenario.name, func(t *testing.T) {
				root := t.TempDir()
				source := filepath.Join(root, "source")
				target := filepath.Join(root, "target")
				results := filepath.Join(root, "results")
				cache := filepath.Join(root, "cache")
				for _, directory := range []string{source, target, results, cache} {
					if err := os.Mkdir(directory, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				for name, content := range scenario.files {
					if err := os.WriteFile(filepath.Join(source, name), []byte(content), 0o600); err != nil {
						t.Fatal(err)
					}
				}

				reporter := &terminalNotificationReporter{}
				app := processing.NewApp(reporter)
				app.Startup(context.Background())
				wantStatus := models.DiagnosticStatusCompleted
				wantNotification := "completed"
				if scenario.cancel {
					wantStatus = models.DiagnosticStatusCancelled
					wantNotification = "cancelled"
				} else if scenario.failReport {
					wantStatus = models.DiagnosticStatusInconclusive
					wantNotification = "error"
				}

				notifications := 0
				reporter.onTerminal = func(notification string) {
					notifications++
					if notification != wantNotification {
						t.Errorf("notification = %q, want %q", notification, wantNotification)
					}
					diagnostics := app.GetScanDiagnostics()
					if diagnostics.Status != wantStatus {
						t.Errorf("status during notification = %q, want %q", diagnostics.Status, wantStatus)
					}
					duplicate := app.GetResults()
					coverage := app.GetMissingContentResult()
					if wantStatus != models.DiagnosticStatusCompleted {
						if duplicate != nil || coverage != nil || len(diagnostics.Diagnostics) != 0 {
							t.Error("unsuccessful execution published results or retained diagnostics")
						}
						return
					}
					if mode == models.ExecutionModeDuplicates {
						if duplicate == nil || coverage != nil {
							t.Errorf("expected only duplicate result during notification, got %#v, %#v", duplicate, coverage)
						} else if duplicate.Diagnostics.Status != wantStatus {
							t.Errorf("duplicate diagnostic status = %q, want %q", duplicate.Diagnostics.Status, wantStatus)
						}
					} else {
						if coverage == nil || duplicate != nil {
							t.Errorf("expected only coverage result during notification, got %#v, %#v", duplicate, coverage)
						} else if coverage.Diagnostics.Status != wantStatus {
							t.Errorf("coverage diagnostic status = %q, want %q", coverage.Diagnostics.Status, wantStatus)
						}
					}
					if (mode == models.ExecutionModeFindMissingContent || scenario.name == "duplicates") && !app.CheckIfResultsExist() {
						t.Error("report is unavailable during completion notification")
					}
				}
				reporter.onProgress = func(title string, percent float64) {
					// Filtering runs after discovery, avoiding cancellation of a
					// discovery progress sender (tracked separately under R3).
					if title != "Filtering" || percent != 0 {
						return
					}
					if scenario.cancel {
						app.CancelExecution()
					}
					if scenario.failReport {
						if err := os.Remove(results); err != nil {
							t.Errorf("remove report directory: %v", err)
						}
					}
				}

				err := app.StartExecution(models.ExecutionParams{
					Mode:                 mode,
					Directories:          []string{source},
					FindMissingSourceDir: source,
					FindMissingTargetDir: target,
					ResultsDir:           results,
					CacheDir:             cache,
					WorkerCount:          1,
					BufferSizeBytes:      1024,
				})
				if scenario.failReport && err == nil {
					t.Error("expected report failure")
				} else if !scenario.failReport && err != nil {
					t.Errorf("execution failed: %v", err)
				}
				if notifications != 1 {
					t.Errorf("terminal notifications = %d, want 1", notifications)
				}
			})
		}
	}
}

type terminalNotificationReporter struct {
	reporting.NoOpReporter
	onTerminal func(string)
	onProgress func(string, float64)
}

func (r *terminalNotificationReporter) FinishExecution(context.Context) {
	r.onTerminal("completed")
}

func (r *terminalNotificationReporter) ReportError(context.Context, string) {
	r.onTerminal("error")
}

func (r *terminalNotificationReporter) LogDetailedStatus(_ context.Context, message string) {
	if message == "Process Stopped." {
		r.onTerminal("cancelled")
	}
}

func (r *terminalNotificationReporter) LogProgress(_ context.Context, title string, percent float64) {
	r.onProgress(title, percent)
}
