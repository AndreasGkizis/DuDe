package processing

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"fmt"
	"reflect"
	"testing"
)

func TestExecutionResultPublisherDerivesDiagnosticsFromResult(t *testing.T) {
	for _, mode := range []string{"duplicates", "coverage"} {
		for _, withDiagnostics := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/diagnostics=%t", mode, withDiagnostics), func(t *testing.T) {
				app := NewApp(reporting.NoOpReporter{})
				publisher := &executionResultPublisher{}
				diagnostics := models.NewScanDiagnostics(models.DiagnosticStatusCompleted)
				if withDiagnostics {
					diagnostics.Diagnostics = append(diagnostics.Diagnostics, models.Diagnostic{
						Path: "/private", Reason: "permission denied",
						Phase: models.DiagnosticPhaseWalking, Scope: models.DiagnosticScopeSource,
					})
				}
				want := copyScanDiagnostics(diagnostics)
				if mode == "duplicates" {
					publisher.PublishDuplicateResult(models.DuplicateResult{Diagnostics: diagnostics})
				} else {
					publisher.PublishMissingContentResult(models.MissingContentResult{Diagnostics: diagnostics})
				}
				// Collection must own its copy, even before application publication.
				if withDiagnostics {
					diagnostics.Diagnostics[0].Path = "/changed"
				}
				publisher.PublishTo(app)

				var resultDiagnostics models.ScanDiagnostics
				if mode == "duplicates" {
					result := app.GetResults()
					if result == nil || app.GetMissingContentResult() != nil {
						t.Fatal("expected only a duplicate result")
					}
					resultDiagnostics = result.Diagnostics
				} else {
					result := app.GetMissingContentResult()
					if result == nil || app.GetResults() != nil {
						t.Fatal("expected only a coverage result")
					}
					resultDiagnostics = result.Diagnostics
				}
				if !reflect.DeepEqual(resultDiagnostics, want) {
					t.Errorf("result diagnostics = %#v, want %#v", resultDiagnostics, want)
				}
				if got := app.GetScanDiagnostics(); !reflect.DeepEqual(*got, want) {
					t.Errorf("standalone diagnostics = %#v, want %#v", *got, want)
				}
			})
		}
	}
}
