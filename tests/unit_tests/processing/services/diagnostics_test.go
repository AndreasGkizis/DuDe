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

func TestDiagnosticSinkCompletedSnapshotIsEmptyArray(t *testing.T) {
	sink := services.NewDiagnosticSink()
	sink.Close()

	got := sink.CompletedSnapshot()
	if got.Status != models.DiagnosticStatusCompleted {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if got.Diagnostics == nil || len(got.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want non-nil empty slice", got.Diagnostics)
	}
}

func TestDiagnosticSinkCompletedSnapshotIsDefensiveAndDoesNotCloseSink(t *testing.T) {
	sink := services.NewDiagnosticSink()
	defer sink.Close()
	for _, path := range []string{"/b", "/a"} {
		if !sink.Submit(services.DiagnosticEvent{
			Diagnostic: models.Diagnostic{Path: path},
		}, services.ErrorDiagnostic) {
			t.Fatalf("submission for %q was rejected", path)
		}
	}

	got := sink.CompletedSnapshot()
	if got.Status != models.DiagnosticStatusCompleted {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if len(got.Diagnostics) != 2 {
		t.Fatalf("retained %d diagnostics, want 2", len(got.Diagnostics))
	}
	if got.Diagnostics[0].Path != "/a" || got.Diagnostics[1].Path != "/b" {
		t.Fatalf("diagnostics are not sorted: %#v", got.Diagnostics)
	}
	got.Diagnostics[0].Path = "/changed"
	if next := sink.CompletedSnapshot(); next.Diagnostics[0].Path != "/a" {
		t.Fatalf("snapshot mutation changed retained diagnostics: %#v", next.Diagnostics)
	}
	if !sink.Submit(services.DiagnosticEvent{
		Diagnostic: models.Diagnostic{Path: "/c"},
	}, services.ErrorDiagnostic) {
		t.Fatal("CompletedSnapshot closed the sink")
	}
	if len(got.Diagnostics) != 2 {
		t.Fatal("later submission changed the earlier snapshot")
	}

	sink.Close()
	if final := sink.CompletedSnapshot(); len(final.Diagnostics) != 3 {
		t.Fatalf("retained %d diagnostics after close, want 3", len(final.Diagnostics))
	}
}
