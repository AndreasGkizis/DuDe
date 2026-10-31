package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCreateHashesSupportsIndependentCollectionsWithSharedCache(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.txt")
	secondPath := filepath.Join(root, "second.txt")
	if err := os.WriteFile(firstPath, []byte("same content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("same content"), 0o600); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	args := models.ExecutionParams{UseCache: true, CacheDir: cacheDir}
	manager := services.NewMemoryManager(&args, 1, 2)
	memory := manager.LoadMemory()
	manager.Start()

	first := fileHashMap(firstPath)
	second := fileHashMap(secondPath)
	hashFilesForTest(t, first, manager, &memory)
	hashFilesForTest(t, second, manager, &memory)
	manager.WaitForCache(context.Background(), func(_, _ int64) {})
	manager.CloseAndWait()

	if hash := loadFileHash(t, first, firstPath).Hash; hash == "" {
		t.Fatal("expected first collection to be hashed")
	}
	if hash := loadFileHash(t, second, secondPath).Hash; hash == "" {
		t.Fatal("expected second collection to be hashed")
	}

	reader := services.NewMemoryManager(&args, 1, 1)
	loaded := reader.LoadMemory()
	reader.Start()
	reader.CloseAndWait()
	if _, ok := loaded[firstPath]; !ok {
		t.Fatal("expected first collection hash in shared cache")
	}
	if _, ok := loaded[secondPath]; !ok {
		t.Fatal("expected second collection hash in shared cache")
	}
}

func TestCreateHashesReturnsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	path := filepath.Join(t.TempDir(), "file.txt")
	files := fileHashMap(path)
	tracker := visuals.NewProgressTracker(ctx, reporting.NoOpReporter{}, "Hashing")
	tracker.Start()
	manager := services.NewMemoryManager(&models.ExecutionParams{}, 1, 1)
	memory := map[string]models.FileHash{}

	err := services.CreateHashes(ctx, files, 1, tracker, manager, &memory, new(int), make(chan error, 1))
	tracker.Wait()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if hash := loadFileHash(t, files, path).Hash; hash != "" {
		t.Fatalf("expected cancelled hashing not to populate a hash, got %q", hash)
	}
}

func TestCreateHashesRejectsZeroWorkers(t *testing.T) {
	ctx := context.Background()
	tracker := visuals.NewProgressTracker(ctx, reporting.NoOpReporter{}, "Hashing")
	tracker.Start()
	manager := services.NewMemoryManager(&models.ExecutionParams{}, 1, 1)

	err := services.CreateHashes(ctx, &sync.Map{}, 0, tracker, manager, &map[string]models.FileHash{}, new(int), make(chan error, 1))
	tracker.Wait()
	if err == nil {
		t.Fatal("expected zero workers to be rejected")
	}
}

func hashFilesForTest(t *testing.T, files *sync.Map, manager *services.MemoryManager, memory *map[string]models.FileHash) {
	t.Helper()
	ctx := context.Background()
	tracker := visuals.NewProgressTracker(ctx, reporting.NoOpReporter{}, "Hashing")
	tracker.Start()
	if err := services.CreateHashes(ctx, files, 1, tracker, manager, memory, new(int), make(chan error, 1)); err != nil {
		t.Fatalf("hash files: %v", err)
	}
	tracker.Wait()
}

func fileHashMap(path string) *sync.Map {
	files := &sync.Map{}
	files.Store(path, models.FileHash{
		FileName: filepath.Base(path),
		FilePath: path,
		FileSize: int64(len("same content")),
	})
	return files
}
