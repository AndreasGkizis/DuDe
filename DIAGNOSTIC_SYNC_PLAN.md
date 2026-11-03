# Asynchronous Diagnostic Synchronization

**Status:** In progress — implementation started; regression tests expose remaining issues
**Decision date:** 2026-11-02

## Summary

Unify filesystem-walk and file-hashing error handling across duplicate and coverage executions without making scan workers perform synchronous logging or frontend publication.

Each execution creates one shared diagnostic sink. Discovery and hashing create structured diagnostic events, pass them through the execution's error-handler callback for classification, and submit them to the sink. The sink retains accepted diagnostics in a lossless, mutex-protected collection and logs every submitted event asynchronously.

The execution flow owns sink shutdown. After all discovery or hashing producers have finished, the flow closes the sink, takes one deterministic snapshot, generates any report, and publishes the result and diagnostics as one atomic application-state update.

This plan deliberately changes the diagnostics and duplicate-result APIs. Duplicate matching semantics remain unchanged.

## Goals

1. Keep scan workers free from synchronous logging, sorting, and frontend work.
2. Use one shared diagnostic pipeline for walking and hashing.
3. Preserve source, target, and duplicate scope.
4. Expose the same diagnostics model in duplicate and coverage results.
5. Publish result, report path, diagnostics, and execution status atomically.
6. Make diagnostics lossless, deterministic, race-free, and configurable internally per run.
7. Preserve duplicate matching, coverage matching, cache behavior, cancellation, and worker cleanup semantics.

## Non-Goals

1. Do not expose error classification policy to users, Wails arguments, the UI, or persistent settings.
2. Do not change eligible filesystem entries or duplicate/coverage matching rules.
3. Do not classify cancellation as a skipped-path diagnostic.
4. Do not convert cache, comparison, reporting, validation, or other fatal execution errors into scan diagnostics.
5. Do not make the frontend assemble, classify, sort, or merge diagnostics.
6. Do not create a report when a completed duplicate run has neither duplicate rows nor diagnostic rows.

## Error Classification

Every error must be logged or reported, but not every error is a scan diagnostic.

### Retained scan diagnostics

These are path-specific, non-fatal failures where processing can continue:

- directory-walk entry failures;
- inability to inspect a directory entry;
- file `Stat` failures during hashing;
- file-read or hash-calculation failures.

### Log/report-only errors

These remain on their existing execution paths and are not retained in `ScanDiagnostics`:

- validation failures;
- cache initialization, cache read, or cache write failures;
- fatal worker/setup failures;
- comparison failures;
- CSV creation, writing, flushing, or closing failures;
- unexpected internal errors;
- cancellation.

Cancellation remains a normal stopped state. It is never a skipped diagnostic.

## Internal Diagnostic Types

The public diagnostic model is generic and applies to all execution modes:

```go
type DiagnosticPhase string

type DiagnosticScope string

const (
    DiagnosticPhaseWalking DiagnosticPhase = "walking"
    DiagnosticPhaseHashing DiagnosticPhase = "hashing"

    DiagnosticScopeSource    DiagnosticScope = "source"
    DiagnosticScopeTarget    DiagnosticScope = "target"
    DiagnosticScopeDuplicate DiagnosticScope = "duplicate"
)

type Diagnostic struct {
    Path   string          `json:"path"`
    Reason string          `json:"reason"`
    Phase  DiagnosticPhase `json:"phase"`
    Scope  DiagnosticScope `json:"scope"`
}
```

The underlying Go error and logging severity remain internal:

```go
type DiagnosticSeverity string

const (
    DiagnosticSeverityWarning DiagnosticSeverity = "warning"
    DiagnosticSeverityError   DiagnosticSeverity = "error"
)

type DiagnosticEvent struct {
    Diagnostic Diagnostic
    Err        error
    Severity   DiagnosticSeverity
}
```

The underlying error is never serialized. Classification can use `errors.Is` and `errors.As`, rather than matching error-message text.

## Error Disposition and Per-Run Configuration

The classification callback is intentionally small and is selected per execution. It is not a public setting and is not stored in `models.ExecutionParams`.

```go
type ErrorDisposition int

const (
    ErrorLogOnly ErrorDisposition = iota
    ErrorDiagnostic
    ErrorFatal
)

type RunOptions struct {
    HandleDiagnostic func(DiagnosticEvent) ErrorDisposition
}
```

A run with a nil `HandleDiagnostic` callback is an internal configuration error and must be rejected before discovery, hashing, or report generation starts. There is no global fallback policy.

The operation-level sequence is explicit:

```go
event := DiagnosticEvent{
    Diagnostic: diagnostic,
    Err:        err,
    Severity:   severity,
}

disposition := options.HandleDiagnostic(event)
sink.Submit(event, disposition)
```

The caller then applies the disposition:

| Disposition | Logged | Retained | Continue |
|---|---:|---:|---:|
| `ErrorLogOnly` | yes | no | yes |
| `ErrorDiagnostic` | yes | yes | yes |
| `ErrorFatal` | yes | no | no |

The sink logs every submitted event asynchronously. Fatal errors are submitted for logging and then returned by the caller so the flow becomes inconclusive.

## Diagnostic Sink

There is one sink per execution:

```go
type DiagnosticSink struct { ... }

func NewDiagnosticSink(...) *DiagnosticSink
func (s *DiagnosticSink) Submit(event DiagnosticEvent, disposition ErrorDisposition)
func (s *DiagnosticSink) Close()
func (s *DiagnosticSink) Snapshot() []Diagnostic
```

The sink must:

1. accept concurrent walk and hash events;
2. asynchronously log every submitted event using the existing logger and preserved severity;
3. retain only `ErrorDiagnostic` events;
4. retain diagnostics in a mutex-protected in-memory collection;
5. never drop diagnostics because a logging queue is full;
6. avoid making workers wait for logging or sorting;
7. close logging exactly once;
8. reject or safely ignore submissions after closure rather than panic;
9. return a defensive, sorted snapshot;
10. tolerate cancellation and early flow returns after all producers have completed.

Diagnostic retention is intentionally not implemented as a bounded channel. A bounded channel could fill when failures outpace logging and would either block workers or drop diagnostics. The sink's collection path is a short mutex-protected append; only logging is asynchronous.

The flow owns shutdown. Discovery and hashing must wait for their own workers before returning. The flow closes the sink only after no producer can call `Submit` again.

## Deterministic Ordering

The final snapshot is sorted by:

1. scope;
2. full path;
3. phase;
4. reason.

The same ordered snapshot is used for:

- result objects;
- `GetScanDiagnostics()`;
- UI rendering;
- coverage reports;
- duplicate reports.

## Public Diagnostics State

```go
type ScanDiagnosticsStatus string

const (
    ScanStatusNotStarted   ScanDiagnosticsStatus = "notStarted"
    ScanStatusRunning      ScanDiagnosticsStatus = "running"
    ScanStatusCompleted    ScanDiagnosticsStatus = "completed"
    ScanStatusCancelled    ScanDiagnosticsStatus = "cancelled"
    ScanStatusInconclusive ScanDiagnosticsStatus = "inconclusive"
)

type ScanDiagnostics struct {
    Status      ScanDiagnosticsStatus `json:"status"`
    Diagnostics []Diagnostic           `json:"diagnostics"`
}
```

`Diagnostics` is always serialized as an empty array rather than `null`.

State rules:

| State | Result objects | Diagnostics |
|---|---|---|
| `notStarted` | empty | `[]` |
| `running` | empty | `[]` |
| `completed` | completed mode result | finalized snapshot, possibly `[]` |
| `cancelled` | empty | `[]` |
| `inconclusive` | empty | `[]` |

Diagnostics collected during cancellation or inconclusive failure are logged but discarded from retained application state.

## Result Models and API

Both mode-specific result models contain the same completed diagnostics snapshot:

```go
type MissingContentResult struct {
    TotalSourceFiles int              `json:"totalSourceFiles"`
    MissingFiles     []MissingSourceFile `json:"missingFiles"`
    ReportPath       string           `json:"reportPath,omitempty"`
    Diagnostics      ScanDiagnostics  `json:"diagnostics"`
}

type DuplicateResult struct {
    Groups      []FileHash      `json:"groups"`
    ReportPath  string          `json:"reportPath,omitempty"`
    Diagnostics ScanDiagnostics `json:"diagnostics"`
}
```

The result and `GetScanDiagnostics()` must contain the same status and ordered diagnostic snapshot for the same completed execution.

API changes:

- `GetResults()` changes from `[]models.FileHash` to `*models.DuplicateResult`.
- `GetResults()` returns `nil` when no completed duplicate result exists.
- `GetMissingContentResult()` continues returning the completed coverage result or `nil`.
- `GetScanDiagnostics()` remains the mode-independent diagnostics/status API and returns the current `ScanDiagnostics` state.

## Atomic Application State

Use one internal state object under the existing application mutex:

```go
type executionState struct {
    DuplicateResult      *models.DuplicateResult
    MissingContentResult *models.MissingContentResult
    ScanDiagnostics      models.ScanDiagnostics
}
```

Every transition replaces the complete state in one lock-protected operation.

At execution start:

```text
clear duplicate result
clear coverage result
set diagnostics status = running
set diagnostics = []
```

On successful completion:

```text
publish mode result and finalized diagnostics atomically
```

On cancellation:

```text
clear results
set diagnostics status = cancelled
set diagnostics = []
```

On inconclusive failure:

```text
clear results
set diagnostics status = inconclusive
set diagnostics = []
```

Full Reset returns the application to the same state as startup:

```text
clear all results and report state
set diagnostics status = notStarted
set diagnostics = []
```

The existing `fullReset` frontend event remains sufficient; no new diagnostics event is required.

## Execution Lifecycle

Both flows use this lifecycle:

```text
validate run options and callback
    ↓
create one sink for the execution
    ↓
discover and hash using scoped diagnostic events
    ↓
wait for all producers
    ↓
close sink and take sorted snapshot
    ↓
match/collect and generate report
    ↓
publish result and diagnostics atomically
```

The sink is finalized before any successful result is made visible. A report-generation failure prevents successful publication and produces an inconclusive state.

## Duplicate Flow

1. Create one shared sink and run-level diagnostic callback.
2. Pass duplicate scope to discovery and hashing events.
3. Preserve duplicate candidate filtering, hashing, matching, paranoid comparison, cache behavior, and result ordering semantics.
4. Finalize diagnostics before collecting the final duplicate result.
5. Build a `DuplicateResult` containing groups, report path, and diagnostics.
6. Do not turn skipped paths into duplicate groups.
7. Render diagnostics in a separate UI section beside duplicate groups.
8. Publish the result and diagnostics atomically.

### Duplicate CSV

Duplicate CSV files are named:

```text
duplicate_results_YYYY_MM_DD_HH_MM_SS.csv
```

No file is created when there are no duplicate groups and no diagnostics. A file is created when either exists, including diagnostics-only runs.

The duplicate CSV columns are:

```text
Type
Filename
Full Path
Duplicate Filename
Duplicate Full Path
Reason
```

Rows use two types:

- `Duplicate`: one row per duplicate relationship;
- `Skipped`: one row per diagnostic.

Duplicate rows contain:

```text
Type       = Duplicate
Reason     = identical content
```

Skipped rows contain the diagnostic path and reason; duplicate-specific columns remain empty. The phase is appended to the reason:

```text
permission denied (phase: walking)
```

Rows are deterministic:

1. `Duplicate` rows before `Skipped` rows;
2. full path;
3. duplicate full path;
4. reason.

Duplicate report creation, writing, flushing, and closing failures are inconclusive. Incomplete files are removed when possible, no successful result is published, and `ReportPath` remains empty.

A diagnostic-only successful run receives a populated `ReportPath`.

## Coverage Flow

1. Create one shared sink and run-level diagnostic callback.
2. Pass source scope to source discovery and hashing.
3. Pass target scope to target discovery and hashing.
4. Preserve target filtering: files whose known sizes cannot match any source file are not hashed, but discovery diagnostics remain visible.
5. Finalize diagnostics before matching and report generation.
6. Build coverage result and diagnostics from the same snapshot.
7. Generate the coverage report only after all diagnostic rows are available.
8. Publish the result and diagnostics atomically.
9. Render diagnostics in a separate UI section beside coverage results.

### Coverage CSV

Coverage retains its existing columns:

```text
Type
Scope
Filename
Full Path
Size (bytes)
Reason
```

Diagnostic rows continue to use `Type = Skipped` and lowercase `Scope = source` or `target`. Their phase is appended only to the existing `Reason` field:

```text
permission denied (phase: walking)
input/output error (phase: hashing)
```

Missing rows retain their existing reason and do not receive a diagnostic phase.

## Report and Results Directory Behavior

Both report types are written to the configured results directory:

```text
<results-dir>/duplicate_results_YYYY_MM_DD_HH_MM_SS.csv
<results-dir>/coverage_report_YYYY_MM_DD_HH_MM_SS.csv

If a path already exists, append `_1`, `_2`, and so on; do not add unnecessary timestamp precision.
```

`ReportPath` identifies the report from the current completed run. `ShowResults()` may continue opening the results directory. `CheckIfResultsExist()` must use the current completed result's `ReportPath` rather than treating stale files from previous executions as the current result.

## Implementation Steps

- [x] **1. Update models and public API contracts.**
  - Add typed diagnostic phase, scope, severity, disposition, status, event, and public models.
  - Add `Diagnostics` to `DuplicateResult` and `MissingContentResult`.
  - Change `GetResults()` to return `*DuplicateResult`.
  - Update Wails bindings and frontend types.
  - Verification: model JSON tests and API contract tests.

- [x] **2. Implement the per-run callback and diagnostic sink.**
  - Add `RunOptions.HandleDiagnostic` and reject nil callbacks before work starts.
  - Implement lossless mutex-protected retention.
  - Implement asynchronous logging for all submitted events with severity preservation.
  - Implement close-once behavior and deterministic snapshots.
  - Verification: concurrent submission, classification, logging, close, post-close, ordering, high-volume, and race tests.

- [x] **3. Integrate discovery.**
  - Replace direct skipped-path slices and direct walk-error logging with structured events.
  - Preserve scope, phase, eligibility, symlink, hidden-file, special-entry, nested-directory, and cancellation behavior.
  - Ensure walk producers finish before the flow closes the sink.
  - Verification: discovery parity, error classification, scope, logging, and cancellation tests.

- [x] **4. Integrate hashing.**
  - Replace flow-specific hash error channels with structured events and the shared sink.
  - Preserve removal of failed files from candidate collections.
  - Preserve cache fallback, progress behavior, cancellation, and worker cleanup.
  - Verification: `Stat` failure, read/hash failure, fatal error, cache fallback, cancellation, high-volume, and cleanup tests.

- [x] **5. Add atomic execution state.**
  - Replace separate result publication with one internal `executionState`.
  - Publish `running`, `completed`, `cancelled`, `inconclusive`, and `notStarted` transitions atomically.
  - Clear all results at execution start and Full Reset.
  - Ensure empty diagnostics serialize as `[]`.
  - Verification: state transition, repeated execution, reset, cancellation, failure, and concurrent getter tests.

- [x] **6. Adapt duplicate execution.**
  - Use one sink and duplicate scope for walk/hash events.
  - Finalize diagnostics before result/report creation.
  - Create duplicate CSV rows and diagnostics-only reports according to the finalized format.
  - Preserve duplicate groups and matching behavior.
  - Verification: duplicate walk failure, hash failure, no-result/no-diagnostic, diagnostics-only, CSV ordering, report failure, API, and regression tests.

- [x] **7. Adapt coverage execution.**
  - Use source and target scopes for discovery and hashing.
  - Remove synchronous flow-specific hash-error draining.
  - Generate coverage diagnostics and CSV rows from the finalized snapshot.
  - Preserve coverage matching, irrelevant-target filtering, report cleanup, cancellation, and inconclusive semantics.
  - Verification: source/target diagnostics, phase formatting, deterministic CSV, report failure, cancellation, and coverage regression tests.

- [ ] **8. Update frontend presentation and bindings.** (Initial implementation complete; mode-coupling cleanup deferred.)
  - Consume `GetResults()` as `DuplicateResult`.
  - Render duplicate and coverage diagnostics in separate sections beside their results.
  - Render status values including `running`, `completed`, `cancelled`, `inconclusive`, and `notStarted`.
  - Clear all previous UI results at execution start and Full Reset.
  - Keep `ShowResults()` opening the results directory.
  - Verification: frontend build, Wails binding generation/build, and manual checks for both modes.

- [x] **9. Complete integration and concurrency verification.**
  - Test mixed walk/hash failures in one run.
  - Test many concurrent failures without deadlock or lost diagnostics.
  - Test repeated executions, cancellation, inconclusive failure, reset, and getters during transitions.
  - Run unit, E2E, race, frontend, and Wails verification commands.

## Compatibility Changes

The following are intentional breaking changes:

1. `SkippedPath` is replaced by public `Diagnostic`.
2. `ScanDiagnostics` changes from source/target fields to status plus generic `Diagnostics`.
3. `DuplicateResult` and `MissingContentResult` gain embedded diagnostics.
4. `GetResults()` changes from `[]models.FileHash` to `*models.DuplicateResult`.
5. Duplicate CSV gains `Type` and `Reason` columns and uses a new filename prefix.

No legacy source/target diagnostic fields or duplicate-result API aliases are required.

## Acceptance Criteria

1. Every submitted walk/hash error is logged asynchronously.
2. Diagnostic retention is lossless under concurrent failure.
3. Retained diagnostics include path, reason, phase, and scope.
4. Classification is selected per run through a required internal callback.
5. The callback can inspect the underlying Go error.
6. One sink implementation is shared by duplicate and coverage flows.
7. Sink shutdown occurs only after all producers finish.
8. Diagnostics are sorted deterministically.
9. Successful results and diagnostics are published atomically.
10. Cancellation and inconclusive executions retain no diagnostics and no results.
11. Startup and Full Reset produce `notStarted` with an empty diagnostic array.
12. A new run clears all previous state and produces `running` with an empty diagnostic array.
13. Duplicate and coverage result objects contain the same diagnostics snapshot exposed by `GetScanDiagnostics()`.
14. Duplicate diagnostics are rendered separately from duplicate groups.
15. Coverage diagnostics are rendered separately from coverage findings.
16. Duplicate CSV files are created only when duplicate or skipped rows exist.
17. Coverage and duplicate diagnostic reasons include phase using the documented suffix format.
18. Report failures produce `inconclusive` state and no successful result.
19. Existing duplicate matching and coverage semantics remain unchanged.
20. Unit, E2E, race, frontend, and Wails verification pass.

## Verification Commands

```bash
GOCACHE=/tmp/dude-go-build go test ./...
GOCACHE=/tmp/dude-go-race go test -race ./...
GOCACHE=/tmp/dude-go-build go test -count=1 ./tests/unit_tests ./tests/e2e_tests
GOCACHE=/tmp/dude-go-race go test -race -count=1 ./tests/unit_tests ./tests/e2e_tests
cd frontend && npm ci --include=dev
cd frontend && npm run build
PATH="$PWD/.tools/bin:$PATH" wails build -f
```

## Current Implementation Status

The implementation is substantially complete. The following parts are present:

- `models.Diagnostic` with typed phase and scope values;
- `models.ScanDiagnostics` with lifecycle status and a generic diagnostic list;
- diagnostic statuses: `notStarted`, `running`, `completed`, `cancelled`, and `inconclusive`;
- `DuplicateResult.Diagnostics` and `MissingContentResult.Diagnostics`;
- atomic internal execution state in `FrontendApp`;
- `GetResults()` returning `*models.DuplicateResult`;
- mode-independent `GetScanDiagnostics()` returning a status-bearing object;
- per-run `RunOptions.HandleDiagnostic` callback;
- internal `DiagnosticEvent` retaining the underlying error and severity;
- one `DiagnosticSink` per duplicate or coverage execution;
- mutex-protected diagnostic retention and asynchronous logging;
- deterministic diagnostic snapshots sorted by scope, path, phase, and reason;
- discovery and hashing integration through the shared sink path, with immediate producer-side submission;
- `DiscoverWithSink()` as the sole discovery API used by production and tests;
- `CreateHashes()` as the sole public hashing API, with internal `runHashWorkers()`;
- recoverable walk and hash path failures retained as warning diagnostics while processing continues;
- duplicate report generation using `duplicate_results_YYYY_MM_DD_HH_MM_SS.csv` with collision suffixes only when needed;
- duplicate `Duplicate` and `Skipped` report rows;
- diagnostic-only duplicate reports;
- phase suffixes in CSV reasons, such as `permission denied (phase: walking)`;
- frontend model, binding, and initial rendering updates;
- updated existing Go and E2E tests for the intentional API/CSV changes.

The remaining deferred work is frontend diagnostic state decoupling; the default handler policy and unbounded logging queue are accepted decisions for now.

## Resolved Issues

The following implementation issues were fixed and covered by regression tests:

1. Duplicate diagnostics now render in a visible container outside the coverage-only summary.
2. Recoverable walk failures use warning severity, are retained as diagnostics, and never stop the scan.
3. Discovery submits events from the walking producer path.
4. Hashing submits events from worker error paths and stops promptly on `ErrorFatal`.
5. JSON and CSV scope values are consistently lowercase.
6. Reports use second-level timestamps with a numeric suffix only when a collision occurs.
7. `CheckIfResultsExist()` verifies that the current report exists and is a regular file.
8. Duplicate report create, write, flush, close, cleanup, and inconclusive publication behavior is tested.

## Additional Risks Not Yet Covered by Automated Regression Tests

### 8. The default handler classifies every submitted path error as a retained diagnostic

`flows.DefaultDiagnosticHandler` currently always returns `ErrorDiagnostic`. The callback mechanism is injectable, but the production default does not distinguish log-only, diagnostic, and fatal categories.

For now, keep the existing default policy: submitted path failures are retained as diagnostics and processing continues. A more differentiated default classification policy is deferred.

### 9. Duplicate and coverage UI state is still coupled

The shared frontend renderer uses `currentCoverageDiagnostics` and the coverage DOM element for duplicate diagnostics. Once a visible duplicate container is added, separate mode-specific state or a generic diagnostics component should be used so mode switching and reset cannot overwrite the wrong presentation.

Add a frontend test or manual verification covering:

- duplicate diagnostics only;
- coverage diagnostics only;
- switching modes after diagnostics;
- Full Reset after diagnostics;
- a new run clearing previous diagnostics.

### 10. Report failure and publication invariants need duplicate-specific tests

Duplicate report failure tests are now covered alongside the existing coverage tests. They cover:

- create failure;
- write failure;
- flush failure;
- close failure;
- incomplete-file removal;
- `inconclusive` state;
- no published duplicate result;
- no published completed diagnostics;
- empty `ReportPath`.

The duplicate report writer tests now cover create, write, flush, close, and incomplete-file cleanup behavior. The duplicate flow E2E test covers inconclusive publication behavior.

### 11. The asynchronous log queue is unbounded

The sink intentionally avoids a bounded channel so workers do not block and diagnostics are not dropped. The current queue can grow without a limit during an extreme failure storm.

This is an accepted design trade-off for now, but future work should measure:

- peak queue size;
- memory usage under high-volume failures;
- logger throughput;
- whether a memory limit or overflow policy is needed.

Do not add dropping or worker-blocking behavior without revisiting the lossless diagnostic requirement.

### 12. Public API compatibility is intentionally broken

The following changes require all consumers to be migrated together:

- `SkippedPath` was replaced by `Diagnostic`;
- source/target diagnostic fields were replaced by generic `ScanDiagnostics.Diagnostics`;
- `GetResults()` changed from `[]FileHash` to `*DuplicateResult`;
- result objects now contain diagnostics;
- duplicate CSV columns and filename changed.

Before release, verify generated Wails bindings and any external API consumers against the new schema.

## Regression Test Inventory

Currently added regression tests:

```text
internal/processing/services/diagnostics_regression_test.go
- TestCreateHashesStopsAfterFatalDisposition
- TestDiscoverWithSinkUsesWarningSeverity
- TestCoverageCSVUsesLowercaseScope
- TestSaveDuplicateReportIncludesDiagnosticOnlyRows
- TestSaveDuplicateReportDoesNotCreateEmptyReport
- TestSaveDuplicateReportUsesUniquePathsForRapidRuns

internal/processing/services/duplicate_report_test.go
- TestSaveDuplicateReportReturnsCreateFailure
- TestSaveDuplicateReportRemovesIncompleteFileAfterWriteFailure
- TestSaveDuplicateReportReturnsCloseFailureAndRemovesFile
- TestWriteDuplicateCSVReportsFlushFailure

tests/unit_tests/diagnostics_state_regression_test.go
- TestCheckIfResultsExistReturnsFalseWhenCurrentReportWasDeleted

tests/unit_tests/frontend_diagnostics_regression_test.go
- TestDuplicateDiagnosticsHaveVisibleContainerOutsideCoverageSummary

tests/e2e_tests/duplicate_flow_regression_test.go
- TestDuplicateFlowSuppressesResultAfterInconclusiveReportFailure
```

The regression tests for fatal handling, CSV scope casing, report-path uniqueness, deleted report detection, frontend visibility, and duplicate report failures are green. Recoverable walk severity is intentionally warning-only. Frontend mode-switch/reset coverage remains deferred.

Existing baseline verification before adding the red regression tests passed:

```text
go test ./...
go test -race ./...
cd frontend && npm run build
PATH="$PWD/.tools/bin:$PATH" wails build -f
```

After fixing the known issues, rerun the complete commands and confirm that the regression tests are green.

## Resume Checklist

When continuing this work:

1. Run the new regression tests first to confirm the current failure set.
2. Fix duplicate diagnostics presentation and add a visible non-coverage container.
3. Preserve warning severity when constructing recoverable walk diagnostic events.
4. Move discovery/hash event submission closer to the producer path.
5. Decide how fatal dispositions cancel active hashing/discovery work.
6. Normalize CSV scope values without changing JSON scope values.
7. Make report filenames collision-safe.
8. Verify report paths with filesystem checks.
9. Add duplicate report failure tests. (Complete.)
10. Add frontend mode-switch/reset diagnostics tests. (Deferred.)
11. Run unit, E2E, race, frontend, and Wails verification. (Complete.)
12. Update the acceptance criteria and mark only completed implementation steps as complete.
