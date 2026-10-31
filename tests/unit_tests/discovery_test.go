package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileDiscoveryCollectsNestedAndHiddenFiles(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "root.txt"),
		filepath.Join(root, "nested", "nested.txt"),
		filepath.Join(root, ".hidden", "hidden.txt"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := services.NewFileDiscovery().Discover(context.Background(), []string{root}, reporting.NoOpReporter{})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	for _, path := range paths {
		value, ok := files.Load(path)
		if !ok {
			t.Fatalf("expected discovered file %q", path)
		}
		if _, ok := value.(models.FileHash); !ok {
			t.Fatalf("expected FileHash for %q, got %T", path, value)
		}
	}
}

func TestFileDiscoveryKeepsRootsSeparateInOneCollection(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	firstFile := filepath.Join(first, "first.txt")
	secondFile := filepath.Join(second, "second.txt")
	for _, path := range []string{firstFile, secondFile} {
		if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := services.NewFileDiscovery().Discover(
		context.Background(),
		[]string{first, second},
		reporting.NoOpReporter{},
	)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	for _, path := range []string{firstFile, secondFile} {
		if _, ok := files.Load(path); !ok {
			t.Fatalf("expected discovered file %q", path)
		}
	}
}

func TestFileDiscoveryReturnsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files, err := services.NewFileDiscovery().Discover(ctx, []string{t.TempDir()}, reporting.NoOpReporter{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if files == nil {
		t.Fatal("expected a file collection even when discovery is cancelled")
	}
}
