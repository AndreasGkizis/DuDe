package unit_tests

// Bug demonstration tests — each test is named after the bug it exposes.
//
// Failing tests (t.Errorf/t.Fatalf) → fail with the current code, pass after the fix.
// Documentation tests (t.Logf)      → always pass but print the observed broken behaviour.
//
// Run all:        go test ./tests/unit_tests/... -v -run TestBug
// Run with race:  go test ./tests/unit_tests/... -v -race -run TestBug

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	database "DuDe/internal/db"
	"DuDe/internal/models"
	"DuDe/internal/models/db_models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

// =============================================================================
// Bug 1 — EnsureDuplicates: local value-copy mutations are never stored back
//
// When EnsureDuplicates detects a false positive (files have the same hash but
// different bytes), it removes the entry from the local copy of FileHash.DuplicatesFound.
// However, it never calls input.Store(itemHash, item) to write that modified copy
// back to the sync.Map.  The sync.Map is therefore left with stale data: false
// positives are still listed as duplicates after the function returns.
//
// Behaviour:  FAILS with current code, passes after fix.
// =============================================================================

func TestBug_EnsureDuplicates_FalsePositivesNotRemovedFromMap(t *testing.T) {
	dir := t.TempDir()

	original := filepath.Join(dir, "original.txt")
	trueDup := filepath.Join(dir, "true_dup.txt")
	falseDup := filepath.Join(dir, "false_dup.txt")

	// original == trueDup (real duplicate), original != falseDup (false positive)
	os.WriteFile(original, []byte("content A"), 0644)
	os.WriteFile(trueDup, []byte("content A"), 0644)
	os.WriteFile(falseDup, []byte("content B — different!"), 0644)

	const fakeHash = "shared_hash"

	m := &sync.Map{}
	m.Store(fakeHash, models.FileHash{
		FileName: "original.txt",
		FilePath: original,
		Hash:     fakeHash,
		DuplicatesFound: []models.FileHash{
			{FileName: "true_dup.txt", FilePath: trueDup, Hash: fakeHash},
			{FileName: "false_dup.txt", FilePath: falseDup, Hash: fakeHash},
		},
	})

	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Test")
	tracker.Start()

	processing.EnsureDuplicates(context.Background(), m, tracker, 4)

	// tracker.Wait ensures all goroutines have incremented — i.e. finished their comparisons.
	tracker.Wait()
	// Extra grace period: wg.Wait() is missing, so goroutines might not have called wg.Done yet.
	time.Sleep(100 * time.Millisecond)

	val, ok := m.Load(fakeHash)
	if !ok {
		// The entry was deleted entirely — meaning both duplicates looked equal at byte level.
		// This shouldn't happen given our files have different content, but is a separate failure mode.
		t.Fatal("BUG (unexpected): entire map entry deleted; expected it to remain with 1 true duplicate")
	}

	fh := val.(models.FileHash)

	// Expected after fix : DuplicatesFound == 1  (only true_dup)
	// Actual with bug    : DuplicatesFound == 2  (both — false_dup was never removed from the map)
	if len(fh.DuplicatesFound) != 1 {
		t.Errorf(
			"BUG CONFIRMED — EnsureDuplicates: false positive NOT removed.\n"+
				"  want  len(DuplicatesFound) = 1 (true_dup only)\n"+
				"  got   len(DuplicatesFound) = %d\n"+
				"  Root cause: 'item' is a value copy inside the goroutine; mutations to\n"+
				"  item.DuplicatesFound are never written back via input.Store(itemHash, item).",
			len(fh.DuplicatesFound),
		)
	}
}

// =============================================================================
// Bug 2 — EnsureDuplicates: wg.Wait() is never called
//
// EnsureDuplicates declares a sync.WaitGroup, calls wg.Add(1) before each
// goroutine, and each goroutine defers wg.Done() — but wg.Wait() is never
// called.  The function therefore returns while goroutines are still running.
// Callers rely on the progress tracker as an implicit sync point, which is
// an invisible coupling and masks the missing wait.
//
// Behaviour:  FAILS with current code (elapsed < 5ms for 8 MB comparison),
//             passes after fix (wg.Wait() blocks until goroutines finish).
// Run with -race to surface any concurrent-access warnings.
// =============================================================================

func TestBug_EnsureDuplicates_ReturnsBeforeGoroutinesFinish(t *testing.T) {
	dir := t.TempDir()

	// Large files so that byte-level comparison takes non-trivial time.
	content := bytes.Repeat([]byte("Z"), 8*1024*1024) // 8 MB
	file1 := filepath.Join(dir, "big1.bin")
	file2 := filepath.Join(dir, "big2.bin")
	os.WriteFile(file1, content, 0644)
	os.WriteFile(file2, content, 0644)

	m := &sync.Map{}
	m.Store("h1", models.FileHash{
		FilePath:        file1,
		Hash:            "h1",
		DuplicatesFound: []models.FileHash{{FilePath: file2, Hash: "h1"}},
	})

	tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Test")
	tracker.Start()

	start := time.Now()
	processing.EnsureDuplicates(context.Background(), m, tracker, 1)
	elapsed := time.Since(start)

	tracker.Wait() // wait for goroutines to actually finish via tracker

	// If wg.Wait() is present the function blocks until the goroutine finishes
	// comparing 8 MB of data — measurably longer than a few milliseconds.
	// Without wg.Wait() it returns almost immediately after launching the goroutine.
	const minExpected = 5 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf(
			"BUG CONFIRMED — EnsureDuplicates returned in %v (too fast).\n"+
				"  Comparing 8 MB of file data should take >%v if the call were blocking.\n"+
				"  Root cause: wg.Wait() is never called, so the function returns before\n"+
				"  goroutines finish — fire-and-forget instead of blocking.",
			elapsed,
			minExpected,
		)
	}
}

// =============================================================================
// Bug 3 — LoadMemory: panics the entire process instead of propagating errors
//
// LoadMemory called common.Must(mm.repo.GetAll()) to unwrap DB results.
// If the database operation fails (e.g. DB closed, corrupt file), the process
// panicked — crashing the whole application — instead of returning the error
// to the caller so it can be handled gracefully.
// common.Must has been removed; LoadMemory now returns (map, error).
//
// Behaviour:  FAILS with current code (panic crashes the test process),
//             passes after fix (error is returned gracefully).
// =============================================================================

func TestBug_LoadMemory_ReturnsErrorInsteadOfPanicking(t *testing.T) {
	db, err := database.InitializeDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("InitializeDatabase: %v", err)
	}
	db.Close() // force all subsequent DB operations to fail

	args := &models.ExecutionParams{UseCache: true, CPUs: 1, BufSize: 100}
	mm := processing.NewMemoryManagerWithDB(args, 100, 1, db)

	_, loadErr := mm.LoadMemory()
	if loadErr == nil {
		t.Error("BUG CONFIRMED — LoadMemory returned nil error on a closed DB; expected an error")
	} else {
		t.Logf("LoadMemory correctly returned error: %v", loadErr)
	}
}

// =============================================================================
// Bug 4 — CreateHashes: unconditional time.Sleep(1 second) on every call
//
// The very first statement in CreateHashes is time.Sleep(1000ms).  It fires
// unconditionally — even when there are zero files to hash.  Every execution
// cycle therefore carries a minimum 1-second tax.
//
// Behaviour:  FAILS with current code (elapsed >= 900ms), passes after fix.
// =============================================================================

func TestBug_CreateHashes_HasUnconditionalOneSleep(t *testing.T) {
	emptyMap := &sync.Map{} // no files — should return almost instantly
	// Use a cancelable context so we can shut down the tracker goroutine after the call.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memory := map[string]models.FileHash{}
	failedCount := 0
	errChan := make(chan error, 10)

	args := &models.ExecutionParams{UseCache: false, CPUs: 1, BufSize: 100}
	mm := processing.NewMemoryManager(args, 100, 1)
	mm.Start()

	tracker := visuals.NewProgressTracker(ctx, reporting.NoOpReporter{}, "Test")
	tracker.Start()

	start := time.Now()
	_ = processing.CreateHashes(ctx, emptyMap, 1, tracker, mm, &memory, &failedCount, errChan)
	elapsed := time.Since(start)

	// Cancel the context so the tracker goroutine exits cleanly (avoids test hang).
	// With 0 files, total==0 so the tracker never self-terminates.
	cancel()
	tracker.Wait()

	// With 0 files there is nothing to do; the function should return in < 50ms.
	// The sleep inflates this to >= 1s regardless of workload.
	const threshold = 900 * time.Millisecond
	if elapsed >= threshold {
		t.Errorf(
			"BUG CONFIRMED — CreateHashes took %v with 0 files.\n"+
				"  Expected: < %v  (nothing to do → return immediately)\n"+
				"  Root cause: time.Sleep(1000 * time.Millisecond) fires unconditionally\n"+
				"  as the very first statement, before any file-count check.",
			elapsed,
			threshold,
		)
	}
}

// =============================================================================
// Bug 5 — Upsert: uses err.Error() string comparison instead of errors.Is()
//
// Upsert calls Update, and if it gets the error string "not found" it falls
// through to Create.  This is brittle: any rename of the error message silently
// breaks the upsert — Create is never called, the record is never persisted,
// and Upsert returns nil (success) despite doing nothing.
//
// Behaviour:  documentation test — always passes, but logs the fragile coupling.
// =============================================================================

func TestBug_Upsert_ReliesOnHardcodedErrorString(t *testing.T) {
	db, err := database.InitializeDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("InitializeDatabase: %v", err)
	}
	defer db.Close()

	repo := database.NewFileHashRepository(db)
	fh := &db_models.FileHash{
		FilePath: "/some/file.txt",
		Hash:     "abc123",
		FileSize: 42,
		ModTime:  "2024-01-01T00:00:00Z",
	}

	// Confirm that Update returns EXACTLY the string "not found" for a missing record.
	// Upsert hard-codes this string: `err.Error() == "not found"`.
	updateErr := repo.Update(fh)
	if updateErr == nil {
		t.Fatal("expected Update to return an error for a missing record")
	}

	t.Logf(
		"BUG DOCUMENTED — Upsert string comparison.\n"+
			"  Update returned error: %q\n"+
			"  Upsert matches this with: err.Error() == \"not found\"\n"+
			"  If the message is ever renamed (e.g. to \"record not found\"), Upsert silently\n"+
			"  skips Create, returns nil, and the record is never saved.\n"+
			"  Fix: use a sentinel error (var ErrNotFound = errors.New(...)) + errors.Is().",
		updateErr.Error(),
	)

	// Show the current path works — only because the string matches exactly.
	if err := repo.Upsert(fh); err != nil {
		t.Fatalf("Upsert (create path via string match) unexpectedly failed: %v", err)
	}
	got, err := repo.GetByPath(fh.FilePath)
	if err != nil || got == nil {
		t.Error("Upsert appeared to succeed but the record was not persisted")
	}
}

// =============================================================================
// Bug 6 — FileHashRepository.Update: real DB error is swallowed, "not found" returned
//
// Update calls GetByPath first.  GetByPath returns (nil, <real error>) when the
// DB is unavailable.  Update checks `if existingFH == nil` BEFORE `if err != nil`,
// so it returns errors.New("not found") instead of the real error.  Upsert then
// sees "not found" and optimistically tries Create, masking the underlying failure.
//
// Behaviour:  FAILS with current code (returns "not found"),
//             passes after fix (returns the real DB error).
// =============================================================================

func TestBug_Update_SwallowsRealDBError(t *testing.T) {
	db, err := database.InitializeDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("InitializeDatabase: %v", err)
	}
	db.Close() // force all subsequent operations to fail with a real DB error

	repo := database.NewFileHashRepository(db)
	fh := &db_models.FileHash{
		FilePath: "/test/file.txt",
		Hash:     "abc",
		FileSize: 100,
		ModTime:  "2024-01-01T00:00:00Z",
	}

	updateErr := repo.Update(fh)
	if updateErr == nil {
		t.Fatal("expected Update to return an error when the DB is closed")
	}

	// Expected after fix : updateErr should be a real DB error (e.g. "sql: database is closed")
	// Actual with bug    : updateErr is errors.New("not found") — real cause discarded
	if updateErr.Error() == "not found" {
		t.Errorf(
			"BUG CONFIRMED — Update.SwallowsRealDBError.\n"+
				"  got error:      %q\n"+
				"  want:           real DB error (e.g. \"sql: database is closed\")\n"+
				"  Root cause: `if existingFH == nil` is checked before `if err != nil`,\n"+
				"  so when GetByPath returns (nil, <real error>), the real error is discarded\n"+
				"  and Update returns errors.New(\"not found\") instead.",
			updateErr.Error(),
		)
	} else {
		t.Logf("Update returned real error (bug may be fixed): %v", updateErr)
	}
}

// =============================================================================
// Bug 7 — FatalWithFuncName: calls logger.Error() instead of logger.Fatal()
//
// All other log helpers (Info, Warn, Error) call the matching zap method.
// FatalWithFuncName, however, calls logger.Error() — so a "fatal" event is
// logged as an ordinary error and execution continues normally.  Code that
// calls FatalWithFuncName expecting a process abort will silently carry on.
//
// NOTE: We cannot call FatalWithFuncName directly in a test because zap's
// SugaredLogger.Fatal calls os.Exit(1) even when the core is a no-op (it only
// suppresses writing, not the exit hook).  We therefore verify the bug by
// inspecting the function source directly.
//
// Behaviour:  FAILS with current code (wrong call found), passes after fix.
// =============================================================================

func TestBug_FatalWithFuncName_CallsErrorNotFatal(t *testing.T) {
	src, err := os.ReadFile("../../internal/common/logger/logger.go")
	if err != nil {
		t.Fatalf("cannot read logger source: %v", err)
	}

	content := string(src)

	// Find the FatalWithFuncName function body.
	const sig = "func FatalWithFuncName"
	idx := -1
	for i := range content {
		if i+len(sig) <= len(content) && content[i:i+len(sig)] == sig {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatal("FatalWithFuncName not found in logger.go — file structure may have changed")
	}

	// Grab a ~500 char window covering the full function body.
	window := content[idx:]
	if len(window) > 500 {
		window = window[:500]
	}

	// Each log helper (Info/Warn/Error/Fatal) ends with:
	//   logger.<Level>(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
	// FatalWithFuncName should end with logger.Fatal but instead uses logger.Error.
	const bugPattern = `logger.Error(fmt.Sprintf("%s()(line:%d)->`
	const fixPattern = `logger.Fatal(fmt.Sprintf("%s()(line:%d)->`

	if bytes.Contains([]byte(window), []byte(bugPattern)) {
		t.Errorf(
			"BUG CONFIRMED — FatalWithFuncName calls logger.Error() instead of logger.Fatal().\n"+
				"  'Fatal' conditions are silently downgraded to ordinary error logs;\n"+
				"  the process is never terminated — defeating the purpose of fatal-level logging.\n"+
				"  Fix: replace `logger.Error(...)` with `logger.Fatal(...)` in FatalWithFuncName.\n\n"+
				"  Source excerpt:\n%s",
			window,
		)
	} else if bytes.Contains([]byte(window), []byte(fixPattern)) {
		t.Log("logger.Fatal is called — bug appears fixed.")
	} else {
		t.Log("Neither expected pattern found; logger source structure may have changed.")
	}
}
