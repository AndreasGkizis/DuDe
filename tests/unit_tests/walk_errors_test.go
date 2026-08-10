package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

func TestWalkDirMissingRootDoesNotPanic(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "removed")
	files := &sync.Map{}

	panicked := runWalkDirAndCapturePanic(missingRoot, files, reporting.NoOpReporter{})

	if panicked {
		t.Fatal("expected a missing root directory to return without panicking")
	}
	if countSyncMapEntries(files) != 0 {
		t.Fatal("expected a missing root directory to produce no files")
	}
}

func TestWalkDirReportsUnreadableDirectoryAndContinuesWithSiblings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are required for this test")
	}

	root := t.TempDir()
	blockedDirectory := filepath.Join(root, "blocked")
	if err := os.Mkdir(blockedDirectory, 0o700); err != nil {
		t.Fatalf("create blocked directory: %v", err)
	}
	blockedFile := filepath.Join(blockedDirectory, "blocked.txt")
	if err := os.WriteFile(blockedFile, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("write blocked file: %v", err)
	}
	accessibleFile := filepath.Join(root, "z-accessible.txt")
	if err := os.WriteFile(accessibleFile, []byte("accessible"), 0o600); err != nil {
		t.Fatalf("write accessible file: %v", err)
	}

	if err := os.Chmod(blockedDirectory, 0); err != nil {
		t.Fatalf("remove directory permissions: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDirectory, 0o700) })
	if _, err := os.ReadDir(blockedDirectory); err == nil {
		t.Skip("current user can still read permissionless directories")
	}

	reporter := &walkErrorReporter{}
	files := &sync.Map{}
	panicked := runWalkDirAndCapturePanic(root, files, reporter)

	if panicked {
		t.Fatal("expected an unreadable child directory to be skipped without panicking")
	}
	if _, exists := files.Load(accessibleFile); !exists {
		t.Fatal("expected scanning to continue with accessible sibling files")
	}
	if _, exists := files.Load(blockedFile); exists {
		t.Fatal("expected files below the unreadable directory to be skipped")
	}
	if !reporter.contains("blocked") {
		t.Fatal("expected the unreadable directory to be reported to the frontend")
	}
}

func runWalkDirAndCapturePanic(path string, files *sync.Map, reporter reporting.Reporter) (panicked bool) {
	counter := visuals.NewProgressCounter(context.Background(), reporter, "Reading", 1)
	counter.Start()

	func() {
		defer func() {
			panicked = recover() != nil
		}()
		processing.WalkDir(context.Background(), path, files, counter)
	}()

	counter.WaitForSenders()
	counter.Wg.Wait()
	return panicked
}

func countSyncMapEntries(values *sync.Map) int {
	count := 0
	values.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

type walkErrorReporter struct {
	messages []string
}

func (*walkErrorReporter) LogProgress(context.Context, string, float64) {}
func (*walkErrorReporter) LogFilesCount(context.Context, int64, int64)  {}
func (*walkErrorReporter) FinishExecution(context.Context)              {}

func (reporter *walkErrorReporter) LogDetailedStatus(_ context.Context, message string) {
	reporter.messages = append(reporter.messages, message)
}

func (reporter *walkErrorReporter) contains(fragment string) bool {
	for _, message := range reporter.messages {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

var _ reporting.Reporter = (*walkErrorReporter)(nil)
