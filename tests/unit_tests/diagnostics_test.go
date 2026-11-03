package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"fmt"
	"sync"
	"testing"
)

func TestDiagnosticSinkRetainsOnlyDiagnosticEventsInDeterministicOrder(t *testing.T) {
	sink := services.NewDiagnosticSink()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			diagnostic := models.Diagnostic{
				Path:   fmt.Sprintf("/file-%03d", 100-i),
				Reason: "permission denied",
				Phase:  models.DiagnosticPhaseWalking,
				Scope:  models.DiagnosticScopeDuplicate,
			}
			sink.Submit(services.DiagnosticEvent{
				Diagnostic: diagnostic,
				Err:        fmt.Errorf("%s", diagnostic.Reason),
				Severity:   services.DiagnosticSeverityWarning,
			}, services.ErrorDiagnostic)
			sink.Submit(services.DiagnosticEvent{
				Diagnostic: models.Diagnostic{Path: diagnostic.Path + ".log-only"},
				Err:        fmt.Errorf("log-only"),
				Severity:   services.DiagnosticSeverityWarning,
			}, services.ErrorLogOnly)
		}(i)
	}
	wg.Wait()
	sink.Close()

	got := sink.Snapshot()
	if len(got) != 100 {
		t.Fatalf("retained %d diagnostics, want 100", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Path >= got[i].Path {
			t.Fatalf("diagnostics are not sorted: %q before %q", got[i-1].Path, got[i].Path)
		}
	}
}

func TestDiagnosticSinkSnapshotIsEmptyArrayAndPostCloseSubmitIsRejected(t *testing.T) {
	sink := services.NewDiagnosticSink()
	sink.Close()

	if sink.Submit(services.DiagnosticEvent{Diagnostic: models.Diagnostic{Path: "/late"}}, services.ErrorDiagnostic) {
		t.Fatal("expected post-close submission to be rejected")
	}
	if got := sink.Snapshot(); got == nil || len(got) != 0 {
		t.Fatalf("snapshot = %#v, want non-nil empty slice", got)
	}
}
