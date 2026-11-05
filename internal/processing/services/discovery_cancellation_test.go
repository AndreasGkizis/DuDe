package services

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiscoveryCancellationBetweenContextCheckAndProgressSend(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counter := visuals.NewProgressCounter(ctx, reporting.NoOpReporter{}, "Reading", 1)
	counter.Start()
	entry := discoveryInfoHookEntry{
		DirEntry: entries[0],
		beforeInfo: func() {
			// Info runs after the callback's initial cancellation check.
			// Stop the receiver before the callback attempts its send.
			cancel()
			counter.Wg.Wait()
		},
	}
	done := make(chan error, 1)
	go func() {
		defer counter.SenderFinished()
		done <- storeFilePathsWithSink(ctx, &sync.Map{}, counter, nil)(path, entry, nil)
	}()

	select {
	case err := <-done:
		counter.Wait()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("callback error = %v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		// Unblock the old unconditional send so a failing regression test
		// still joins its producer instead of leaking the goroutine.
		select {
		case <-counter.Channel:
		case <-done:
		}
		counter.Wait()
		t.Fatal("discovery callback blocked sending progress after cancellation")
	}
}

func TestDiscoverWithSinkFatalErrorCancelsOtherRoots(t *testing.T) {
	root := t.TempDir()
	activeRoot := filepath.Join(root, "active")
	if err := os.Mkdir(activeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(activeRoot, name), []byte("content"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	missingRoot := filepath.Join(root, "missing")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sink := NewDiagnosticSink()
	defer sink.Close()
	reporter := &discoveryWaitForCancellationReporter{started: make(chan struct{}), stopped: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := NewFileDiscovery().DiscoverWithSink(ctx, []string{activeRoot, missingRoot}, reporter, sink, func(event DiagnosticEvent) ErrorDisposition {
			// Hold the fatal error until the progress receiver has started.
			select {
			case <-reporter.started:
			case <-ctx.Done():
			}
			return ErrorFatal
		}, models.DiagnosticScopeDuplicate)
		done <- err
	}()

	select {
	case err := <-done:
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) || pathErr.Path != missingRoot || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("discovery error = %v, want missing-root error", err)
		}
		select {
		case <-reporter.stopped:
		default:
			t.Fatal("discovery returned before its progress receiver stopped")
		}
		if diagnostics := sink.Snapshot(); len(diagnostics) != 0 {
			t.Fatalf("fatal cancellation retained skipped-path diagnostics: %#v", diagnostics)
		}
	case <-ctx.Done():
		t.Fatal("fatal discovery error did not stop all roots promptly")
	}
}

// Block progress consumption until cancellation so active walkers cannot rely
// on a receiver remaining available when the fatal root cancels discovery.
type discoveryWaitForCancellationReporter struct {
	reporting.NoOpReporter
	started chan struct{}
	stopped chan struct{}
}

func (r *discoveryWaitForCancellationReporter) LogProgress(ctx context.Context, _ string, _ float64) {
	close(r.started)
	<-ctx.Done()
	close(r.stopped)
}

type discoveryInfoHookEntry struct {
	fs.DirEntry
	beforeInfo func()
}

func (e discoveryInfoHookEntry) Info() (fs.FileInfo, error) {
	e.beforeInfo()
	return e.DirEntry.Info()
}
