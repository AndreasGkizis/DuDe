package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

func TestEnsureDuplicatesPersistsPartialRemoval(t *testing.T) {
	directory := t.TempDir()
	primaryPath := writeParanoidTestFile(t, directory, "primary.txt", "same")
	equalPath := writeParanoidTestFile(t, directory, "equal.txt", "same")
	differentPath := writeParanoidTestFile(t, directory, "different.txt", "nope")

	groups := &sync.Map{}
	groups.Store("hash", models.FileHash{
		FilePath: primaryPath,
		DuplicatesFound: []models.FileHash{
			{FilePath: equalPath},
			{FilePath: differentPath},
		},
	})

	if err := runEnsureDuplicatesAndWait(t, groups); err != nil {
		t.Fatalf("ensure duplicates: %v", err)
	}

	value, exists := groups.Load("hash")
	if !exists {
		t.Fatal("expected group with one confirmed duplicate to remain")
	}
	duplicates := value.(models.FileHash).DuplicatesFound
	if len(duplicates) != 1 {
		t.Fatalf("expected one confirmed duplicate, got %d", len(duplicates))
	}
	if duplicates[0].FilePath != equalPath {
		t.Fatalf("expected %q to remain, got %q", equalPath, duplicates[0].FilePath)
	}
}

func TestEnsureDuplicatesDoesNotConfirmComparisonErrors(t *testing.T) {
	directory := t.TempDir()
	primaryPath := writeParanoidTestFile(t, directory, "primary.txt", "same")
	missingPath := filepath.Join(directory, "missing.txt")

	groups := &sync.Map{}
	groups.Store("hash", models.FileHash{
		FilePath: primaryPath,
		DuplicatesFound: []models.FileHash{
			{FilePath: missingPath},
		},
	})

	if err := runEnsureDuplicatesAndWait(t, groups); err == nil {
		t.Fatal("expected comparison failure to be returned")
	}

	if _, exists := groups.Load("hash"); exists {
		t.Fatal("expected a duplicate that could not be compared to be removed")
	}
}

func TestEnsureDuplicatesCompletesProgressWhenPrimaryFileCannotOpen(t *testing.T) {
	directory := t.TempDir()
	missingPrimaryPath := filepath.Join(directory, "missing-primary.txt")
	duplicatePath := writeParanoidTestFile(t, directory, "duplicate.txt", "same")

	groups := &sync.Map{}
	groups.Store("hash", models.FileHash{
		FilePath: missingPrimaryPath,
		DuplicatesFound: []models.FileHash{
			{FilePath: duplicatePath},
		},
	})

	if err := runEnsureDuplicatesAndWait(t, groups); err == nil {
		t.Fatal("expected primary file open failure to be returned")
	}
}

func runEnsureDuplicatesAndWait(t *testing.T, groups *sync.Map) error {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	tracker := visuals.NewProgressTracker(ctx, reporting.NoOpReporter{}, "Comparing")
	tracker.Start()
	err := processing.EnsureDuplicates(ctx, groups, tracker, 1)

	done := make(chan struct{})
	go func() {
		tracker.Wait()
		close(done)
	}()

	select {
	case <-done:
		cancel()
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("timed out waiting for paranoid comparison progress to complete")
	}

	return err
}

func writeParanoidTestFile(t *testing.T, directory, name, contents string) string {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
	return path
}
