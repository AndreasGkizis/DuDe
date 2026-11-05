# DuDe Project Context

This is a handoff for future work. It describes the source tree after the PR #10 review fixes, including commit `1206ca5`; it does not describe a newly published release. Inspect the implementation before relying on historical assumptions.

## Project Summary

**DuDe — Duplicate Detection** is a Go/Wails desktop application with two execution modes:

- **Duplicates:** find equivalent file content across one or more selected directories.
- **Find Missing Content:** identify source files whose content is absent from a target directory, regardless of filename or relative path. One target file may cover multiple identical source files.

Both modes share discovery, size filtering, hashing, optional SQLite caching, diagnostics, and report services. Paranoid Mode adds byte-for-byte verification. DuDe reports findings but does not delete files.

## Current State

- Repository: `https://github.com/AndreasGkizis/DuDe`.
- Review branch: `unified-diagnostic-sink`, targeting `master`.
- Latest tag observed: `v1.2.1`; source metadata remains `1.2.0`, overridden from the tag by release packaging.
- Go 1.24, Wails v2.11.0, pure-Go `modernc.org/sqlite`, and Zap logging.
- Plain JavaScript/HTML/CSS with Vite 3 and generated Wails bindings.
- Frontend behavioral tests use Node's built-in test runner, JSDOM, and Vite module loading; there is no frontend application framework.
- Supported release targets: Linux AMD64 and Windows AMD64. macOS is not validated or supported.
- Linux desktop builds require GTK3 and WebKitGTK 4.1; `wails.json` configures `webkit2_41`.
- License: Apache License 2.0.

## Review Completion and Validation Boundary

The agreed PR #10 review implementation is complete:

- R1–R7 resolved: simulator removal, terminal notification ordering, cancellation-safe discovery progress, duplicate-result buttons, exclusive report creation, consistent sink synchronization, and complete CI test discovery.
- S1–S3 resolved: dead cleanup removal, result-owned completed diagnostics, and deep copies of nested duplicate results.
- S4: keep the existing diagnostic handler and default classification policy as-is.
- Logging-queue memory investigation and E2E fixture isolation are general backlog items in root `TODO.md`, not remaining implementation steps in the diagnostic plan.

Recorded local checks passed: all Go tests with race detection, frontend behavioral tests and production build, Go vet/build, workflow actionlint, and diff checks. Frontend tests/build were also checked under Node 18.20.8 when updating CI.

This is not full release validation: a fresh Wails desktop build and interactive smoke testing were not performed during these review fixes. The GitHub-hosted CI result is not inferred from local checks. npm audit reports six existing Vite-toolchain vulnerabilities (one moderate, five high); those dependencies were not upgraded as part of the review.

See `DIAGNOSTIC_SYNC_PLAN.md` for design and review evidence, `development.md` for commands, and `TODO.md` for the general backlog.

## Important Decisions

1. **Keep MD5 for the current trust model.** It supports local, non-adversarial content comparison, not authentication or signatures. Paranoid Mode provides byte verification.
2. **Size filtering only excludes impossible matches.** Equal size does not establish equivalent content.
3. **Keep SQLite pure Go.** `modernc.org/sqlite` avoids a CGO SQLite dependency in self-contained releases.
4. **Caching is optional and non-critical.** Cache failures warn and disable caching rather than stop scanning.
5. **Allow only one active execution per application instance.** `ExecutionGate` coordinates start, cancellation, and Full Reset. Exclusive report creation is defensive hardening, not parallel-run support.
6. **Use one progress bar.** Phase names belong in Current Job; updates are throttled rather than emitted to the frontend per file.
7. **Flows decide outcomes by returning.** The runner publishes terminal application state before notifying the frontend.
8. **Completed diagnostics belong to the concrete result.** There is no separate `PublishScanDiagnostics` method. The runner derives standalone completed diagnostic state from that result.
9. **Keep the injectable handler policy as-is.** Production currently classifies submitted walk/hash path failures as `ErrorDiagnostic`; do not add speculative classification layers.
10. **Keep diagnostic retention lossless.** The asynchronous logging queue is unbounded by an accepted trade-off. Measure before changing its policy; any dropping or backpressure change requires an explicit decision.
11. **Use regression tests for correctness fixes.** Tests under `tests/` remain useful for public behavior; colocated Go tests are now also used under `internal/processing` and `internal/processing/services` for private contracts and service regressions. Run `./...` so neither location is skipped.
12. **Do not run benchmarks unless requested.** Normal tests/builds are expected validation; timing benchmarks are a separate workflow.
13. **Keep changes small and homogeneous.** Prefer standard-library primitives, explicit ownership, and existing abstractions. No file deletion feature without a separate safety design.

## Repository Layout

| Path | Responsibility |
| --- | --- |
| `main.go` | Embeds `frontend/dist`, creates the Wails app, binds `FrontendApp`. |
| `internal/processing/runner.go` | Frontend API, execution orchestration, atomic state publication, defensive copies. |
| `internal/processing/execution_gate.go` | Rejects overlapping runs and coordinates reset/cancellation. |
| `internal/processing/flows/` | Mode-specific validation, registry, duplicate and coverage execution. |
| `internal/processing/services/discovery.go` | Concurrent filesystem discovery with scoped diagnostics. |
| `internal/processing/services/hashing.go` | Hashing, duplicate grouping, and byte verification. |
| `internal/processing/services/candidates.go` | Shared file-size filtering helpers. |
| `internal/processing/services/coverage.go` | Source-to-target content matching. |
| `internal/processing/services/diagnostics.go` | Per-run retention and asynchronous diagnostic logging. |
| `internal/processing/services/memory.go` | Optional asynchronous SQLite hash-cache lifecycle. |
| `internal/processing/services/reports.go` | CSV output, report cleanup, and collision-safe file creation. |
| `internal/db/` | SQLite schema and file-hash repository. |
| `internal/handlers/validation/` | Shared defaults and filesystem validation. |
| `internal/reporting/`, `internal/visuals/` | Reporter interface, Wails events, and progress tracking. |
| `internal/models/` | Execution parameters, results, file metadata, and diagnostics. |
| `internal/common/` | Platform/filesystem helpers, constants, and logging. |
| `frontend/src/` | UI, templates, styling, pagination, and event handlers. |
| `frontend/tests/` | Behavioral tests with mocked Wails calls and actual frontend rendering. |
| `tests/unit_tests/`, `tests/e2e_tests/` | Public-contract regression and filesystem pipeline tests. |
| `tests/benchmarks/` | Explicit production hashing/grouping benchmarks and historical comparisons. |
| `.github/workflows/ci.yml` | Frontend checks plus every Go test package, race detection, vet, and build. |
| `.github/workflows/release.yml` | Tagged Linux/Windows Wails builds and packaging. |
| `package_builder.sh` | Local packaging helper. |

## FrontendApp API and State

Key Wails methods:

- `StartExecution(args)` resolves shared defaults, validates the selected mode, acquires the execution gate, and runs the flow.
- `CancelExecution()` cancels the active execution context.
- `FullReset()` cancels/awaits active work, truncates cache where possible, resets transient state, and emits `fullReset`.
- `GetResults()` returns `*models.DuplicateResult`, or nil without a successful duplicate result.
- `GetMissingContentResult()` returns `*models.MissingContentResult`, or nil without a successful coverage result.
- `GetScanDiagnostics()` returns mode-independent lifecycle status and diagnostics, including when no successful result exists.
- `CheckIfResultsExist()` checks the current completed result's `ReportPath` and requires a regular file; stale reports do not count.
- `ShowResults()` opens the configured results directory, not the CSV directly.
- `RevealInExplorer(path)` opens a result file's containing directory.
- `SelectFolder()` opens a native directory picker; `Startup(ctx)` stores the lifecycle context/platform.

`stateMu` protects execution state. Getters return defensive copies. Duplicate copying recursively clones `FileHash.DuplicatesFound` slices at collection, application-publication, and getter boundaries.

Diagnostic lifecycle statuses are `notStarted`, `running`, `completed`, `cancelled`, and `inconclusive`. A new accepted run clears prior results and diagnostics. Validation/gate rejection leaves existing execution state untouched. Cancellation and execution failure publish no successful result and an empty retained diagnostic array. Full Reset restores `notStarted`.

## Execution Parameters and Validation

`models.ExecutionParams` uses the current names:

| Field | Meaning |
| --- | --- |
| `Mode` | Explicit operation: duplicates (`1`) or find-missing-content (`2`); zero is invalid. |
| `Directories` | Duplicate-mode scan roots. |
| `FindMissingSourceDir`, `FindMissingTargetDir` | Coverage-mode input directories. |
| `UseCache`, `CacheDir` | Enable optional caching and select the `memory.db` directory. |
| `ResultsDir` | CSV output directory. |
| `ParanoidMode` | Enable byte-for-byte verification. |
| `WorkerCount` | Concurrent hashing/comparison worker limit. |
| `BufferSizeBytes` | Memory-manager buffer budget; retain the established API name. |
| `DebugMode` | Enable file logging; independent of debug build symbols. |

Worker counts default/clamp to `runtime.GOMAXPROCS(0)`. Buffer values default to 1024 and clamp to 1,048,576. Blank results/cache paths resolve to safe platform defaults. Results must be writable; input readability validation is mode-specific. Writable probes use temporary files and do not overwrite existing data. Cache unavailability is tolerated.

## Execution Lifecycle

1. The runner resolves/validates arguments and acquires `ExecutionGate`.
2. It publishes `running`, then invokes the selected flow with a temporary result collector.
3. The flow initializes optional logging/cache, creates a diagnostic sink, and performs discovery/filtering/hashing/matching.
4. Producers and progress trackers finish before flow return, including hashing error/cancellation paths. Discovery's blocking progress send selects on `ctx.Done()`.
5. The flow closes the sink, obtains `CompletedSnapshot()`, generates any required report, and publishes one concrete result to the collector.
6. On successful return, the runner atomically publishes the result plus diagnostics derived from it, then calls `FinishExecution`.
7. On error/cancellation, the runner publishes `inconclusive`/`cancelled` before the corresponding notification. The gate is released during final cleanup.

An empty or no-match scan can complete successfully; it still publishes an empty concrete result and receives the runner's completion notification. Report failure prevents successful publication.

### Duplicate Mode

Walk selected roots concurrently with `filepath.WalkDir`, collect metadata into a `sync.Map`, and use `RemoveUniqueSizeFiles` to discard unique sizes. `CreateHashes` re-stats files, reuses valid cached hashes, or calculates MD5. Failed path candidates are removed and diagnosed. `FindDuplicatesInMap` transforms path entries into hash groups. Paranoid Mode uses `EnsureDuplicates` for byte verification and fails visibly on comparison errors. The flow collects groups and finalizes report/result data.

### Find Missing Content Mode

Discover source and target separately with their own diagnostic scopes. Filter hashing candidates by possible size matches across the collections. Preserve unmatched source sizes as missing; do not unnecessarily hash irrelevant target files. Remove skipped paths before `MatchCoverage`. Matching uses content rather than names/paths; target matches are reusable. Coverage generates a report even when the source is empty.

### Diagnostics and Cache

`Diagnostic` contains `path`, `reason`, `phase` (`walking`/`hashing`), and `scope` (`duplicate`/`source`/`target`). Snapshots sort by scope, path, phase, then reason and expose non-nil diagnostic arrays.

`DiagnosticSink` uses one mutex for its closed flag, records, and log queue. Concurrent `Submit` either accepts/queues the event and retains it according to disposition, or rejects without mutation. `Close` rejects further events and waits for every accepted log outside the lock. `CompletedSnapshot` only wraps a defensive snapshot with completed status; it neither closes the sink nor determines execution success.

`ErrorLogOnly` logs without retention; `ErrorDiagnostic` also retains; `ErrorFatal` logs and lets discovery/hashing cancel active work and return the original error. All flows require a handler; production intentionally returns `ErrorDiagnostic` for submitted path failures. Cache, comparison, validation, and report errors are not retained scan diagnostics.

`MemoryManager` loads cached hashes keyed by full path and asynchronously writes new hashes. Validity uses path, size, and modification time; unchanged metadata can conceal changed content. Cache errors disable caching with warnings instead of aborting a scan. SQLite uses one `memory.db` and the `file_hashes` table, with upserts; only `sql.ErrNoRows` becomes the domain not-found error.

## Reports and Runtime Files

- Duplicate reports: `duplicate_results_YYYY_MM_DD_HH_MM_SS.csv`; created only for duplicate or diagnostic rows, including diagnostic-only runs.
- Coverage reports: `coverage_report_YYYY_MM_DD_HH_MM_SS.csv`; created for every completed coverage run.
- Collisions add `_1`, `_2`, etc. Both report writers use exclusive file creation and retry only `os.ErrExist`; existing reports cannot be truncated by a colliding creation attempt.
- CSV includes a UTF-8 BOM; Windows uses semicolons, other platforms commas. Write/flush/close errors propagate, and incomplete output is removed where possible.
- Duplicate columns: `Type`, `Filename`, `Full Path`, `Duplicate Filename`, `Duplicate Full Path`, `Reason`.
- Coverage columns: `Type`, `Scope`, `Filename`, `Full Path`, `Size (bytes)`, `Reason`.
- Diagnostic rows use `Skipped`; reasons include `(phase: walking)` or `(phase: hashing)`. Scope values are lowercase.
- Cache: `memory.db`. Optional debug logs: `logs/YYYY-MM-DD_HH-MM-SS.log` beside the executable.
- Blank cache/results paths default to the executable directory on supported Linux/Windows platforms. macOS helper code is not a support guarantee.

## Frontend Contract

Wails events remain `progressUpdate`, `filesCount`, `detailedLog`, `errorUpdate`, `executionFinished`, and `fullReset`. `executionFinished` is emitted only after completed state is readable. The frontend fetches the mode-specific result; coverage also fetches standalone diagnostics.

Duplicate groups render as paginated, collapsible cards; missing files use coverage cards. Page sizes are 3, 5, 10, 25, and 50. Diagnostics have a shared visible container outside the coverage-only summary. Duplicate report access is enabled from the result's report path, and clear-results is available for groups or diagnostics. Report-button rendering is path-based; the backend existence getter separately checks the actual file.

Progress updates are throttled to roughly 250 ms plus phase boundaries. Unknown/zero progress is indeterminate. Full Reset clears frontend state after backend cleanup. Mode-specific diagnostic state decoupling remains general frontend cleanup, not an unfinished review fix.

## Development, Testing, and CI

See `development.md` for setup. Normal Wails debug builds run real scans (`wails build -debug`); the separate `debug_progress` simulator/build tag was removed. `DebugMode` logging remains available.

CI uses one sequential Ubuntu job with Go from `go.mod` and Node 18, matching the existing release workflow. Build frontend assets before recursive Go checks because `main.go` embeds the ignored `frontend/dist` directory:

```bash
npm --prefix frontend ci --include=dev
npm --prefix frontend test
npm --prefix frontend run build
go test -race -v -count=1 ./...
go vet ./...
go build ./...
```

Run separate E2E-bearing commands sequentially until fixture isolation is addressed; their shared `./test_files/` cleanup can interfere across processes. Colocated tests exercise sink closure/admission, cancellation, report collisions, terminal publication, and deep-copy contracts. Public unit/E2E tests cover matching, validation, cache, CSV, reset, and filesystem behavior. Frontend tests execute actual rendering with mocked Wails bindings.

For a complete desktop build use `wails build -f`; normal `go build ./...` and a Vite build do not replace desktop integration or interactive validation.

Benchmarks remain manual. `BenchmarkHashAndGroupPipeline` exercises `RemoveUniqueSizeFiles`, `CreateHashes`, and `FindDuplicatesInMap`; commands/results are in `tests/benchmarks/README.md`. Do not invoke timing benchmarks unless requested.

The release workflow builds Windows/Linux archives on pushed `v*` tags using Wails v2.11.0, Go 1.24, Node 18, and Linux WebKitGTK 4.1. No release was created by these review fixes.

## Remaining Work and Future Changes

Use root `TODO.md` as the general backlog, rather than reopening the completed diagnostic plan. It includes logging-memory measurement, E2E fixture isolation, Wails/toolchain upgrades, frontend state cleanup, large-file testing, and possible future platform/feature work. Existing frontend toolchain security findings also need remediation. Full desktop build/smoke testing remains the release-validation boundary noted above.

Keep fixes focused, add a reproducing test where practical, and run relevant checks before finalizing. Preserve the agreed content-matching semantics, graceful cache degradation, serialized execution ownership, state-before-notification ordering, defensive result copies, and lossless diagnostic requirement. Do not introduce deletion, new classification policies, or logging overflow behavior without explicit agreement.
