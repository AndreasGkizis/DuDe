# DuDe Project Context

This document is a handoff for future ChatGPT/Codex conversations. It describes the current project state, architecture, execution flow, important decisions, development workflow, and remaining work.

Last reviewed against the repository on **2026-08-18**.

## Project Summary

**DuDe — Duplicate Detection** is a desktop application that finds duplicate files across one or more selected directories.

The application:

1. Walks every selected directory concurrently.
2. Collects file path, size, and modification-time metadata.
3. Removes files whose sizes are unique because they cannot have duplicates in the current scan.
4. Hashes the remaining candidates, reusing valid cached hashes when enabled.
5. Groups files with matching hashes.
6. Optionally verifies hash matches byte for byte in Paranoid Mode.
7. Displays duplicate groups in the UI and writes a CSV report.

DuDe reports duplicates but does not delete files. Deletion is intentionally left to the user.

## Current State

- Repository: `https://github.com/AndreasGkizis/DuDe`
- Branch: `master`
- Latest Git tag at review time: `v1.2.1`
- Source metadata version: `1.2.0`; the release workflow replaces it with the pushed tag version.
- Supported release targets: Linux AMD64 and Windows AMD64.
- macOS is not currently supported or validated.
- License: Apache License 2.0.
- The application is considered functionally close to a production release, with additional cleanup and platform testing listed in `TODO.md`.

## Technology Stack

### Backend

- Go 1.24, declared by `go.mod`
- Wails v2.11.0
- `modernc.org/sqlite` for a pure-Go SQLite implementation
- Zap for optional structured debug logging

### Frontend

- Plain JavaScript, HTML, and CSS
- Vite 3
- Wails-generated JavaScript bindings
- No frontend framework and currently no frontend unit-test framework

### Linux Runtime and Build Dependencies

- GTK3
- WebKitGTK 4.1
- The `webkit2_41` build tag is configured in `wails.json`.

## Important Project Decisions

These decisions came from the development discussions and should not be changed casually.

1. **MD5 remains the file checksum.** It is used for fast, local, non-adversarial file comparison, not passwords, signatures, or security. Paranoid Mode provides optional byte-for-byte verification. Do not replace MD5 solely because it is cryptographically broken.
2. **Size filtering does not claim same-size files are duplicates.** A unique size proves a file cannot have a duplicate in the current scan. Files sharing a size remain candidates and are still hashed.
3. **The SQLite implementation must remain pure Go.** `modernc.org/sqlite` is intentional so releases remain self-contained executables without a CGO SQLite dependency.
4. **The cache is optional and non-critical.** Cache initialization, reads, writes, or closure failures must be logged/reported, but must never stop duplicate detection.
5. **There is one local cache database.** Database changes should be justified for this single-user, local-desktop workload rather than designed as if it were a multi-user server.
6. **Tests belong under `/tests`.** Do not add Go test files beside production packages unless this decision is explicitly revisited.
7. **Use tests first for correctness fixes.** Add a failing regression test, then implement the fix.
8. **Do not run benchmarks unless explicitly requested.** Normal tests and builds may be run for validation.
9. **The UI uses one progress bar.** Each operation resets and reuses it. The operation name belongs in the existing Current Job field, not above the bar.
10. **Keep user-facing documentation concise.** Avoid unnecessary marketing language or long explanations in `README.md`.

## Repository Layout

| Path | Responsibility |
| --- | --- |
| `main.go` | Embeds the frontend, creates the Wails app, and binds `FrontendApp`. |
| `internal/processing/runner.go` | Main application API and end-to-end execution orchestration. |
| `internal/processing/files.go` | Directory walking, size filtering, CSV output, and result-file checks. |
| `internal/processing/hashing.go` | Hashing, duplicate grouping, Paranoid Mode comparison, and result flattening. |
| `internal/processing/memory.go` | Optional asynchronous SQLite cache lifecycle. |
| `internal/processing/execution_gate.go` | Prevents overlapping runs and coordinates cancellation and Full Reset. |
| `internal/processing/execution_mode_*.go` | Selects real or simulated execution through build tags. |
| `internal/db/` | SQLite schema and file-hash repository. |
| `internal/handlers/validation/` | Argument defaults and filesystem validation. |
| `internal/reporting/` | Backend-to-frontend reporting abstraction and Wails implementation. |
| `internal/visuals/` | Progress tracking and throttled UI progress updates. |
| `internal/models/` | Execution parameters, file metadata, duplicate groups, and CSV row models. |
| `internal/common/` | Filesystem helpers, platform helpers, constants, and logging. |
| `frontend/src/` | Plain JavaScript UI, HTML template, CSS, pagination, and Wails event handlers. |
| `tests/unit_tests/` | Regression, validation, database, cache, progress, CSV, and processing tests. |
| `tests/e2e_tests/` | Filesystem-based tests that call the real backend execution pipeline. |
| `tests/benchmarks/` | Production hashing/grouping benchmark and saved before/after results. |
| `.github/workflows/release.yml` | Tagged Linux and Windows release builds. |
| `package_builder.sh` | Local Linux/Windows packaging helper. |
| `development.md` | Development, debugging, testing, benchmark, and build commands. |
| `TODO.md` | Remaining work and completed hardening items. |

## Backend Entry Point

`main.go` embeds `frontend/dist`, constructs a `reporting.WailsReporter`, creates `processing.FrontendApp`, and binds it to Wails.

The Wails window has a 1000x800 baseline, an 800x600 minimum, and Windows-specific Mica configuration.

## FrontendApp API

The following methods are exposed to the frontend through Wails:

- `Startup(ctx)` stores the Wails context and detected platform.
- `SelectFolder()` opens a native directory picker.
- `StartExecution(args)` validates arguments and runs duplicate detection.
- `CancelExecution()` cancels the active execution context.
- `FullReset()` cancels and awaits active work, truncates the cache, resets backend state, and tells the frontend to reset.
- `GetResults()` returns duplicate groups from the last completed execution.
- `ShowResults()` opens the results directory with the platform file manager.
- `RevealInExplorer(path)` opens the containing directory for a result file.
- `CheckIfResultsExist()` checks whether any `results*.csv` file exists in the configured results directory.

`FrontendApp` protects mutable execution arguments, context, and last results with `stateMu`.

## Execution Parameters

`models.ExecutionParams` contains:

| Field | Meaning |
| --- | --- |
| `Directories` | One or more directories to scan. |
| `UseCache` | Enables the SQLite hash cache. |
| `CacheDir` | Directory containing `memory.db`. |
| `ResultsDir` | Directory for generated CSV reports. |
| `ParanoidMode` | Enables byte-for-byte verification after hash grouping. |
| `CPUs` | Maximum concurrent hashing/comparison workers. |
| `BufSize` | Cache-channel buffer size. |
| `DebugMode` | Enables file logging. |

Validation and defaults are applied before state is published:

- At least one readable scan directory is required.
- The results directory must be writable.
- Blank cache and results directories fall back to a safe platform directory.
- `CPUs <= 0` becomes `runtime.GOMAXPROCS(0)` and values above it are clamped.
- `BufSize <= 0` becomes 1024 and values above 1,048,576 are clamped.
- An unavailable cache directory is tolerated because caching is optional.
- Writable checks use temporary probe files and do not overwrite existing user files.

## Production Execution Flow

The real pipeline is orchestrated by `startExecution` in `internal/processing/runner.go`.

### 1. Execution Ownership

`ExecutionGate.Start` rejects a second execution while one is running and rejects starts while Full Reset is active. It creates a cancellable execution context.

All execution-owned workers and channels are awaited before the gate is released.

### 2. Logging and Error Drain

Debug logging is initialized from `DebugMode`. Logging is a no-op when disabled.

An execution error channel is drained by a dedicated goroutine. It records per-file hashing errors without blocking workers and is closed and awaited during cleanup.

### 3. Cache Initialization

`MemoryManager` is created before scanning:

- If caching is disabled, it performs no database work.
- If initialization fails, it records one cache error and disables itself.
- Cached records are loaded into a map keyed by full file path.
- A single asynchronous writer consumes newly calculated hashes.
- Cache failures produce a warning and processing continues without cache.

### 4. Reading

Each selected directory is walked in its own goroutine using `filepath.WalkDir`.

Every discovered non-directory entry is stored in a `sync.Map` as a `FileHash` containing:

- filename
- full path
- byte size
- modification time formatted as RFC3339

Reading progress initially has an unknown total. The Files Checked field shows the discovered count. Directory-walk permission and filesystem errors are reported and skipped where possible instead of silently disappearing or stopping the whole scan.

The final Files Checked value is the number of readable, discovered non-directory entries. It can be lower than a file manager's displayed count because unreadable entries, failed metadata reads, or differing counting rules are not included.

### 5. Filtering

`FilterHashCandidatesBySize` counts how many files have each size and removes only files whose size occurs once.

This is a safe exclusion:

- unique size: cannot be duplicated in this scan, so hashing is unnecessary
- repeated size: might or might not be duplicated, so it remains for hashing

The initial discovered file count is retained for the final Files Checked display even when files are filtered before hashing.

### 6. Hashing and Cache Reuse

`CreateHashes` processes candidates concurrently with a semaphore capped by `CPUs`.

For each candidate:

1. The file is statted again to get current size and modification time.
2. A cached hash is reused when path, size, and modification time match.
3. Otherwise the file is read and an MD5 hash is calculated.
4. Newly calculated metadata and hash are queued for asynchronous cache storage.
5. Files that cannot be statted or read are removed from the candidate map and reported.

Hashing and cache writes overlap. The UI changes to a separate Caching phase only when writes remain after hashing completes, so Caching may be short or absent.

### 7. Finding

`FindDuplicatesInMap` groups files by hash. Hashes represented by only one file are removed. For every repeated hash, the first file becomes the group's primary file and the rest become `DuplicatesFound`.

The map is transformed from `path -> FileHash` into `hash -> duplicate group` during this phase.

### 8. Comparing, Optional

When Paranoid Mode is enabled, `EnsureDuplicates` compares every candidate duplicate byte for byte against its group's primary file.

- Comparison workers are capped by `CPUs`.
- The primary file is opened once per group and rewound between candidates.
- Only confirmed byte-equal files remain in `DuplicatesFound`.
- Comparison errors are not treated as matches.
- Groups with no confirmed duplicates are removed.
- All workers are awaited before the phase completes.

This phase protects against hash collisions among files that were grouped together. MD5 remains acceptable for the normal local workflow by explicit project decision.

### 9. Collecting

The final duplicate groups are copied into `FrontendApp.lastResults`. `GetResults()` later returns a defensive slice copy to the UI.

### 10. CSV Output and Completion

Duplicate groups are flattened into rows containing:

- primary filename
- primary full path
- duplicate filename
- duplicate full path

CSV output includes a UTF-8 BOM for Excel compatibility. Linux and macOS use commas; Windows uses semicolons. Flush and close errors are returned, and incomplete files are removed.

The backend then reports the original discovered file total, emits Done at 100%, and emits `executionFinished`.

## Progress and UI Event Contract

The backend uses a `Reporter` interface so processing can be tested without Wails.

| Event | Payload | Frontend behavior |
| --- | --- | --- |
| `progressUpdate` | `{title, percent}` | Updates Current Job and resets/reuses the single progress bar when the phase changes. |
| `filesCount` | `{current, total}` | Updates Files Checked for every operation. |
| `detailedLog` | string | Shows a detailed status message in Current Job. |
| `errorUpdate` | string | Marks the active phase as failed and displays the error. |
| `executionFinished` | none | Restores controls, marks completion, fetches groups, and renders results. |
| `fullReset` | none | Resets all frontend form and result state. |

Progress updates from `ProgressTracker` are limited to a 250 ms ticker plus immediate phase start/completion events. This avoids emitting one UI event per processed file during hashing, grouping, and comparison.

The Reading phase uses `ProgressCounter`, also updating on a 250 ms ticker, because its total is unknown until walking finishes.

The UI intentionally shows one progress bar, not a stack of phase bars. A zero-percent active phase is displayed as indeterminate until measurable progress arrives.

## Frontend Results

After `executionFinished`, the frontend calls `GetResults()` and maps backend `FileHash` values into frontend duplicate-group models.

Results are rendered as collapsible cards with buttons to reveal each file's directory. Pagination is handled entirely in the frontend.

Available groups-per-page values are:

- 3
- 5
- 10
- 25
- 50

Changing page size preserves the first visible group's approximate position.

## Cancellation and Full Reset

Cancellation propagates through the execution context. Directory walking, worker spawning, hash grouping, and byte comparison check this context.

`ExecutionGate` guarantees:

- only one execution runs at a time
- reset cannot overlap a new execution
- reset cancels and waits for an active execution
- concurrent reset callers wait for the reset owner
- execution state is not cleared while workers still use it

Full Reset then:

1. Resolves the cache directory.
2. Opens and truncates `file_hashes` when possible.
3. Clears execution arguments and in-memory results.
4. Emits `fullReset` for the frontend.

Cache reset failures are warnings rather than fatal application errors.

## SQLite Cache

The cache file is named `memory.db` and contains one table:

```sql
CREATE TABLE IF NOT EXISTS file_hashes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE,
    hash TEXT,
    size INTEGER,
    modified_time TEXT,
    updated_at TEXT,
    created_at TEXT
)
```

Writes use SQLite `INSERT ... ON CONFLICT(path) DO UPDATE`. Database errors are returned directly; only `sql.ErrNoRows` becomes the domain-specific not-found error.

The cache validity check uses path, size, and modification time. A file changed without either metadata value changing can retain a stale cached hash; this is an inherent limitation of the current cache key strategy.

## Runtime Files

- CSV reports: `results_YYYY_MM_DD_HH_MM_SS.csv`
- Cache database: `memory.db`
- Debug logs: `logs/YYYY-MM-DD_HH-MM-SS.log`

When cache/results directories are blank:

- Linux and Windows use the executable directory.
- macOS helper code uses `~/Documents/DuDe`, but macOS releases are not validated or supported.

Debug logs are written beside the executable only when Debug Logging is enabled.

## Error-Handling Policy

- Argument validation errors stop execution and are shown in the UI.
- A second concurrent start is rejected visibly.
- Cancellation is treated as a normal stop rather than a processing failure.
- Cache failures are warnings and never stop scanning.
- Individual directory-walk and hash-read failures are reported and skipped.
- Paranoid comparison errors fail verification rather than creating false matches.
- CSV creation, flush, or close failures fail execution and remove incomplete output where possible.
- All execution-owned channels and goroutines must be closed and awaited on every exit path.

## Testing

All Go tests are under `tests/`.

### Unit Tests

Coverage includes:

- validation and defaults
- safe writable-directory probing
- directory-walk error reporting and file counting
- unique-size filtering
- same-size files with different contents
- hashing read failures
- zero-work and all-failed progress completion
- Paranoid Mode correctness and comparison failures
- SQLite schema and repository behavior
- cache initialization, read, write, and closure failures
- CSV flush failures and incomplete-file cleanup
- execution cancellation/reset synchronization
- frontend-facing validation error reporting

Run without Go's test-result cache:

```bash
go test -count=1 ./tests/unit_tests
```

### End-to-End Tests

The E2E tests create real temporary directory trees and call `FrontendApp.StartExecution`, exercising the production backend pipeline and CSV output without launching the Wails window.

They cover single and multiple directories, nested content, empty folders, hidden files, special characters, cross-directory duplicates, different content, Paranoid Mode, Unicode normalization, and permission failures.

```bash
go test -count=1 ./tests/e2e_tests
```

Run all Go tests uncached:

```bash
go test -count=1 ./...
```

Useful additional checks:

```bash
go test -race -count=1 ./tests/unit_tests
go vet ./...
npm --prefix frontend run build
```

Fast tests are normal. Go tests use small temporary files and in-process SQLite databases, so speed does not mean they were skipped. `-count=1` guarantees results are not reused from the test cache.

## Benchmarks

`BenchmarkHashAndGroupPipeline` executes the real production functions:

- `FilterHashCandidatesBySize`
- `CreateHashes`
- `FindDuplicatesInMap`

It creates 128 real files and tests scenarios where 0% through 100% of files share a size.

Do not run benchmarks unless explicitly requested.

```bash
go test ./tests/benchmarks -run '^$' \
  -bench '^BenchmarkHashAndGroupPipeline$' \
  -benchmem -benchtime=1s -count=10
```

Save before/after output and compare it with `benchstat`. Existing results demonstrated large improvements when most files had unique sizes, with a small overhead when every file remained a candidate.

## Development and Debugging

Install Wails:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

Run in development:

```bash
wails dev
```

Build:

```bash
wails build
```

Force dependency installation and a fresh Wails build:

```bash
wails build -f
```

The Wails frontend install command is `npm ci --include=dev` because Vite is a development dependency but is still required during production compilation.

### Build-Tag Debug Modes

VS Code tasks prompt for an execution mode:

- `real_execution`: normal production pipeline
- `debug_progress`: simulated phase progress without scanning files

The simulated mode is compiled through the `debug_progress` Go build tag. It exists to debug UI phase transitions and progress behavior independently from filesystem speed.

Command-line example:

```bash
wails build -debug -tags debug_progress
```

## Release Process

The GitHub Actions release workflow builds:

- Windows AMD64 ZIP
- Linux AMD64 tarball

It uses Go 1.24, Node.js 18, Wails v2.11.0, and WebKitGTK 4.1 on Linux. A pushed `v*` tag creates a GitHub release and uploads both platform packages.

Example next patch release:

```bash
git tag -a v1.2.2 -m "Release v1.2.2"
git push origin v1.2.2
```

The local packaging script derives its version from the latest tag or accepts `VERSION` explicitly.

## Major Hardening Completed

The project was reviewed and improved throughout the development conversation. Important completed work includes:

1. Database errors are no longer mistaken for missing records.
2. SQL schema, repository, and lifecycle functionality has dedicated tests.
3. File metadata is captured during directory walking.
4. Unique-size files are excluded before hashing.
5. Production hashing/grouping benchmarks and before/after comparison instructions were added.
6. Result groups per page can be changed dynamically.
7. CSV flush and close failures are detected and incomplete files are removed.
8. Writable-directory checks no longer modify a pre-existing probe-named file.
9. Progress phases complete explicitly, including zero-work and all-failed phases.
10. Reading, filtering, hashing, caching, finding, comparing, and collecting are represented as distinct work phases.
11. The UI reuses one progress bar and displays phase names in Current Job.
12. Files Checked updates during all operations and returns to the original discovered total at completion.
13. Every execution-owned worker and channel is closed and awaited.
14. Full Reset is synchronized with active execution cleanup.
15. Processing and validation failures are reported clearly in the UI.
16. Cache initialization and runtime failures disable caching safely without stopping duplicate detection.
17. Paranoid Mode waits for all workers and never treats comparison errors as confirmed duplicates.
18. Release metadata, runtime files, dependencies, and supported platforms were documented.

## Remaining Work

Current `TODO.md` items, in their present order:

1. Remove unused code.
2. Review log levels and details.
3. Test with large files and constrained buffers.
4. Throttle or summarize repeated filesystem errors before sending them to the UI.
5. Simplify frontend state flow.
6. Validate macOS permissions, packaging, signing, notarization, and runtime behavior on real hardware.
7. Consider a duplicate-file deletion feature.

Additional notes retained in `TODO.md`:

- Investigate whether metadata such as time and size should be packed into one value/blob for simpler or faster handling.
- Reconsider use of a `Must` helper pattern.

The repeated-filesystem-error item matters because a large number of inaccessible files can still flood the Wails event stream and make the UI stutter, despite normal progress updates already being throttled.

## Guidance for Future ChatGPT/Codex Work

When continuing this project:

1. Read `README.md`, `development.md`, `TODO.md`, and this file first.
2. Inspect the current implementation before relying on historical assumptions in this document.
3. Keep changes focused and consistent with the existing architecture.
4. For bug fixes, write the failing test first and place it under `tests/`.
5. Run the smallest relevant tests first, then broader tests before finalizing.
6. Do not run benchmarks unless the user asks.
7. Preserve MD5 unless the application's trust model changes.
8. Preserve `modernc.org/sqlite` and the self-contained executable requirement.
9. Cache problems must degrade gracefully and never stop duplicate detection.
10. Avoid high-frequency frontend events; progress should stay throttled or summarized.
11. Keep the single progress-bar design and Current Job phase label.
12. Treat the initial discovered file count as the final Files Checked total.
13. Keep macOS support marked experimental/unsupported until tested on real hardware.
14. Do not add deletion behavior without explicit user approval and a careful safety design.
15. Keep README changes short; put implementation detail in `development.md` or this context file.

## Terminology

- **Reading:** walking selected directories and collecting metadata.
- **Filtering:** removing files with sizes that occur only once.
- **Hashing:** calculating or reusing checksums for remaining candidates.
- **Caching:** finishing asynchronous writes of newly calculated hashes.
- **Finding:** grouping files by checksum and removing unique hashes.
- **Comparing:** optional byte-for-byte verification in Paranoid Mode.
- **Collecting:** copying final duplicate groups into frontend-accessible state.
- **Duplicate group:** one primary file plus one or more files with matching content criteria.
- **Files Checked:** phase-specific progress while running; original discovered file total when complete.
