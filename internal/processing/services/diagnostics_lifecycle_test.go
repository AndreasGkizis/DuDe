package services

import (
	"DuDe/internal/models"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiagnosticSinkConcurrentSubmitAndClose(t *testing.T) {
	sink := NewDiagnosticSink()
	defer sink.Close()
	const producers, events = 32, 32
	accepted := make([]bool, producers*events)
	ready := make(chan struct{}, producers)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for producer := 0; producer < producers; producer++ {
		workers.Add(1)
		go func(producer int) {
			defer workers.Done()
			for event := 0; event < events; event++ {
				index := producer*events + event
				accepted[index] = sink.Submit(DiagnosticEvent{
					Diagnostic: models.Diagnostic{Path: fmt.Sprintf("/event-%04d", index)},
				}, ErrorDiagnostic)
				if event == 0 {
					ready <- struct{}{}
					<-start
				}
			}
		}(producer)
	}
	for producer := 0; producer < producers; producer++ {
		<-ready
	}
	closed := make(chan []models.Diagnostic, 1)
	go func() {
		<-start
		sink.Close()
		closed <- sink.Snapshot()
	}()
	close(start)
	workers.Wait()
	atClose := <-closed
	final := sink.Snapshot()
	if !reflect.DeepEqual(atClose, final) {
		t.Error("retained diagnostics changed after Close returned")
	}
	var want []string
	for index, ok := range accepted {
		if ok {
			want = append(want, fmt.Sprintf("/event-%04d", index))
		}
	}
	sort.Strings(want)
	got := make([]string, len(final))
	for index, diagnostic := range final {
		got[index] = diagnostic.Path
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("retained diagnostics do not match accepted submissions: got %d, want %d", len(got), len(want))
	}
}

func TestDiagnosticSinkConcurrentCloseWaitsForAllAcceptedLogs(t *testing.T) {
	sink := NewDiagnosticSink()
	t.Cleanup(sink.Close)
	started := make(chan struct{})
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	// logEvent evaluates the original error when Reason is empty. Block that
	// evaluation to test shutdown without replacing the application's logger.
	if !sink.Submit(DiagnosticEvent{Err: diagnosticLifecycleError(func() string {
		close(started)
		<-release
		return "blocked log event"
	})}, ErrorLogOnly) {
		t.Fatal("initial submission rejected")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("asynchronous logging did not start")
	}

	var logged atomic.Int32
	for _, disposition := range []ErrorDisposition{ErrorLogOnly, ErrorDiagnostic, ErrorFatal} {
		if !sink.Submit(DiagnosticEvent{
			Diagnostic: models.Diagnostic{Path: fmt.Sprintf("/disposition-%d", disposition)},
			Err: diagnosticLifecycleError(func() string {
				logged.Add(1)
				return "queued log event"
			}),
		}, disposition) {
			t.Fatal("submission rejected before closure")
		}
	}
	const closers = 8
	var workers sync.WaitGroup
	for i := 0; i < closers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			sink.Close()
		}()
	}
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()
	<-sink.stop
	select {
	case <-done:
		t.Fatal("Close returned before the in-flight log completed")
	default:
	}
	if sink.Submit(DiagnosticEvent{Diagnostic: models.Diagnostic{Path: "/rejected"}}, ErrorDiagnostic) {
		t.Error("submission accepted after closure began")
	}
	unblock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not finish after logging was released")
	}
	if got := logged.Load(); got != 3 {
		t.Fatalf("logged %d queued events, want 3", got)
	}
	retained := sink.Snapshot()
	if len(retained) != 1 || retained[0].Path != fmt.Sprintf("/disposition-%d", ErrorDiagnostic) {
		t.Fatalf("unexpected retained diagnostics: %#v", retained)
	}
	if len(sink.logQueue) != 0 {
		t.Fatal("logging queue was not drained")
	}
}

type diagnosticLifecycleError func() string

func (e diagnosticLifecycleError) Error() string { return e() }
