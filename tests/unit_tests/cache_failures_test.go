package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	database "DuDe/internal/db"
	"DuDe/internal/models"
	"DuDe/internal/processing"
)

func TestMemoryManagerDisablesCacheWhenInitializationFails(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(cachePath, []byte("file"), 0o600); err != nil {
		t.Fatalf("create invalid cache path: %v", err)
	}

	manager := processing.NewMemoryManager(&models.ExecutionParams{
		UseCache: true,
		CacheDir: cachePath,
	}, 1, 1)

	_, _, enabled := manager.CacheProgress()
	if enabled {
		t.Fatal("expected cache initialization failure to disable caching")
	}
}

func TestMemoryManagerLoadFailureDoesNotPanic(t *testing.T) {
	cacheDirectory := t.TempDir()
	manager := processing.NewMemoryManager(&models.ExecutionParams{
		UseCache: true,
		CacheDir: cacheDirectory,
	}, 1, 1)
	dropCacheTable(t, cacheDirectory)

	var cachedFiles map[string]models.FileHash
	panicked := func() (panicked bool) {
		defer func() {
			panicked = recover() != nil
		}()
		cachedFiles = manager.LoadMemory()
		return false
	}()

	if panicked {
		t.Fatal("expected cache read failure to return an empty cache without panicking")
	}
	if len(cachedFiles) != 0 {
		t.Fatalf("expected an empty cache after read failure, got %d records", len(cachedFiles))
	}
	_, _, enabled := manager.CacheProgress()
	if enabled {
		t.Fatal("expected cache read failure to disable caching")
	}
}

func TestMemoryManagerWriteFailureDisablesCacheAndDrainsQueue(t *testing.T) {
	cacheDirectory := t.TempDir()
	manager := processing.NewMemoryManager(&models.ExecutionParams{
		UseCache: true,
		CacheDir: cacheDirectory,
	}, 2, 1)
	dropCacheTable(t, cacheDirectory)

	manager.Push(cacheTestFile("/files/first.txt", "first-hash"))
	manager.Push(cacheTestFile("/files/second.txt", "second-hash"))
	manager.Start()
	manager.SenderFinished()

	done := make(chan struct{})
	go func() {
		manager.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the failed cache writer to drain its queue")
	}

	completed, total, enabled := manager.CacheProgress()
	if enabled {
		t.Fatal("expected a cache write failure to disable further cache writes")
	}
	if completed != 0 {
		t.Fatalf("expected failed writes not to count as completed, got %d", completed)
	}
	if total != 2 {
		t.Fatalf("expected both queued writes to be tracked, got %d", total)
	}
}

func TestDisabledMemoryManagerPerformsNoCacheWork(t *testing.T) {
	manager := processing.NewMemoryManager(&models.ExecutionParams{}, 1, 1)
	manager.Start()
	manager.Push(cacheTestFile("/files/file.txt", "hash"))
	manager.SenderFinished()
	manager.WaitForCache(context.Background(), func(_, _ int64) {
		t.Fatal("expected disabled caching not to report progress")
	})

	completed, total, enabled := manager.CacheProgress()
	if enabled || completed != 0 || total != 0 {
		t.Fatalf("expected disabled cache state, got enabled=%v completed=%d total=%d", enabled, completed, total)
	}
}

func dropCacheTable(t *testing.T, cacheDirectory string) {
	t.Helper()

	db, err := database.GetDatabaseConnection(cacheDirectory)
	if err != nil {
		t.Fatalf("open cache database: %v", err)
	}
	if _, err := db.Exec("DROP TABLE file_hashes"); err != nil {
		_ = db.Close()
		t.Fatalf("drop cache table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close cache database: %v", err)
	}
}

func cacheTestFile(path, hash string) models.FileHash {
	return models.FileHash{
		FileName: filepath.Base(path),
		FilePath: path,
		Hash:     hash,
		FileSize: 10,
		ModTime:  "2026-08-10T12:00:00Z",
	}
}
