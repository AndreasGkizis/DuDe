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

The initial implementation is present, but the PR review below identifies outstanding correctness and verification work. Treat the review checklist as the current resume point, rather than the historical completion notes below. Frontend diagnostic state decoupling remains deferred; the default handler policy and unbounded logging queue remain accepted decisions pending explicit reconsideration.

## Resolved Issues

The following implementation issues were fixed and covered by regression tests:

1. Duplicate diagnostics now render in a visible container outside the coverage-only summary.
2. Recoverable walk failures use warning severity, are retained as diagnostics, and never stop the scan.
3. Discovery submits events from the walking producer path.
4. Hashing submits events from worker error paths and stops promptly on `ErrorFatal`.
5. JSON and CSV scope values are consistently lowercase.
6. Reports use second-level timestamps with a numeric suffix for collisions. Exclusive file creation now prevents overwriting existing reports; see R5 below.
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

## PR #10 Review Findings and Current Resume Checklist

Reviewed head: `21da4b8` (`unified-diagnostic-sink`). Review verdict: request changes. This section supersedes the historical resume checklist and completion notes for outstanding work. No implementation fixes were made during the review.

Work through the items individually, adding a focused regression test with each fix. Keep solutions based on standard Go primitives and avoid adding abstractions without a concrete need.

### R1 — Resolved: remove the unused `debug_progress` build

- [x] Remove the progress simulator and the production flow's conditional build tag.
- [x] Remove simulator selection from VS Code and Zed tasks; retain normal Wails debug builds.
- [x] Update development documentation to describe normal debug builds.
- [x] Verify after removal: `go build ./...`, `go test -count=1 ./...`, and `go vet ./...` passed. Editor task JSON validation and `git diff --check` also passed. A full Wails desktop build was not run.

Decision: remove the development-only simulator rather than maintain a second duplicate-flow implementation. The application's `DebugMode` logging option is unaffected.

Follow-up simplification: removed the flow-local `completedDiagnosticsFromSnapshot` helper and use `sink.CompletedSnapshot()` in both flows. This convenience method wraps the deterministic, defensive `Snapshot()` with completed status; it does not close the sink or determine execution success. Closure remains explicit in the flows. Tests cover completed status, a non-nil empty collection, sorted defensive copies, and submissions after taking a completed snapshot. Removed the duplicate flow's unnecessary reconstruction of an already-completed diagnostic value. Terminal notification ordering remains separate work under R2.

Locations: `internal/processing/flows/duplicate_execution.go:198` and `internal/processing/flows/find_missing_content.go:146` at the reviewed head.

The coverage flow calls a helper defined in a file excluded by `//go:build !debug_progress`. Confirmed failure:

```text
go build -tags debug_progress ./...
internal/processing/flows/find_missing_content.go:146:16: undefined: completedDiagnosticsFromSnapshot
```

Acceptance: normal builds and tests pass without a simulator-specific configuration. No CI check for the removed build tag is needed. The failure above is retained as historical review evidence.

### R2 — Resolved: publish terminal state before notifications

- [x] Make the runner own the terminal completion notification after successful state publication.
- [x] Remove premature completion notifications from production flows. The debug simulator was removed under R1.
- [x] Publish cancelled/inconclusive state before sending the corresponding terminal notification.
- [x] Add a reporter test that reads getters immediately during the terminal notification.
- [x] Wait for hashing progress trackers on error/cancellation paths before returning from either flow.

Locations: `internal/processing/flows/duplicate_execution.go:110,191`, `internal/processing/flows/find_missing_content.go:167`, and `internal/processing/runner.go:284–294` at the reviewed head.

At the reviewed head, flows emitted `executionFinished` before their collected result was published by the runner. The frontend could therefore fetch a nil result or diagnostics still marked `running`. In the duplicate no-candidates branch, completion was even emitted before diagnostic report creation, which could subsequently fail. The notification-ordering problem predated parts of this PR, but undermined the new atomic-publication guarantee.

Implementation: `StartExecution` now publishes completed state before calling `FinishExecution`, publishes inconclusive state before `ReportError`, and publishes cancelled state before the `Process Stopped.` status message. Flows determine their outcome by returning a result or error; they no longer send completion notifications. Empty duplicate scans now also receive the runner's completion notification. Validation and gate rejections still leave existing execution state untouched.

Regression coverage: `tests/unit_tests/execution_notification_test.go`, `TestExecutionPublishesStateBeforeTerminalNotification`, exercises both modes with empty input, unique sizes, same-size different content, duplicate content, report failure, and cancellation. Reporter callbacks synchronously check terminal state, mode-specific results, report availability where expected, and exactly one terminal notification. The test failed before the fix and passed afterward.

Race verification also exposed hashing progress trackers outliving cancelled flows. Both flows now wait for those trackers on error paths as well as successful paths, preventing late progress/logging after terminal publication.

Acceptance verified: successful flow return → completed state publication → completion notification. Failed report generation emits no successful completion. Getters invoked during terminal notifications observe the corresponding terminal state.

Verification: `go test -race -count=1 ./...`, the focused notification test repeated 25 times under `-race`, `go build ./...`, `go vet ./...`, and `git diff --check` passed. Interactive Wails UI behavior was not exercised. Discovery cancellation remains separate work under R3.

### R3 — Resolved: make discovery progress sends cancellation-aware

- [x] Make the discovery progress send select on `ctx.Done()`.
- [x] Add cancellation-during-progress and multi-root fatal-disposition regression tests.

Location: `internal/processing/services/discovery.go:130` at the reviewed head.

At the reviewed head, the unconditional `pt.Channel <- 1` could block forever if cancellation stopped the progress receiver after the callback's initial context check. Discovery then waited forever for its senders. A fatal disposition from one root could trigger this while another root was discovering files. The unsafe send existed previously; fatal-disposition cancellation added another trigger.

Implementation: the blocking send now selects on `ctx.Done()` and returns `ctx.Err()` when cancelled. The initial cancellation check and normal discovery behavior remain unchanged; no new production abstraction was introduced.

The original temporary review reproduction has been replaced by permanent regression tests in `internal/processing/services/discovery_cancellation_test.go`:

- `TestDiscoveryCancellationBetweenContextCheckAndProgressSend` cancels during `DirEntry.Info()` and waits for the progress receiver to exit before allowing the callback to send. It failed before the fix and passes afterward. Its failure path unblocks the old send and joins the test producer rather than leaking it.
- `TestDiscoverWithSinkFatalErrorCancelsOtherRoots` exercises an active root alongside a missing root, holding the progress receiver until cancellation. It verifies discovery returns the original missing-root error (not context cancellation), joins its progress receiver, and retains no fatal/cancellation diagnostics.

The progress send now uses:

```go
select {
case pt.Channel <- 1:
    return nil
case <-ctx.Done():
    return ctx.Err()
}
```

Acceptance verified: cancellation and fatal discovery errors return promptly, with walkers and progress goroutines joined. The existing `reading.Wait()` still joins discovery producers and the receiver before discovery returns.

Verification: both targeted tests passed 100 repetitions under `-race`; `go test -race -count=1 ./...`, `go build ./...`, `go vet ./...`, and `git diff --check` passed.

### R4 — Resolved: restore duplicate report access and diagnostic-only clearing

- [x] Set the duplicate report button state from the published result/report path, consistently with coverage rendering.
- [x] Enable clearing results for diagnostic-only duplicate runs.
- [x] Add behavioral frontend tests for groups with a report, diagnostics only with a report, and no report.

Location: `frontend/src/main.js:838`, `renderResults`, at the reviewed head.

At the reviewed head, execution startup disabled `showResultsButton`, but duplicate rendering never re-enabled it. `refreshResultsButtonState()` only ran at initialization. Diagnostic-only rendering also returned without enabling the clear-results control. The existing HTML substring test could not detect these behavioral failures.

Implementation: `renderResults` sets report access from `reportPath`/`ReportPath`, consistently with coverage rendering, and sets clear-results availability from the presence of groups or diagnostics before any early return. The duplicate-group list stays hidden when there are no groups; diagnostics remain in their existing independent container. No backend behavior changed.

Regression coverage: `frontend/tests/duplicate-results.test.mjs` loads the real frontend through Vite, runs the execution-finished listener with mocked Wails bindings, and inspects button states and DOM output using JSDOM. Six scenarios cover groups and diagnostics with/without reports, empty results, and null results. Tests also exercise clearing the rendered output. Report-access and diagnostic-only clear tests failed before the fix and pass afterward.

Frontend tests use Node.js's built-in test runner (`npm --prefix frontend test`) and a Node 18-compatible JSDOM development dependency. `development.md` documents installation and execution. CI integration remains under R7.

Acceptance verified: completed duplicate reports enable report access, diagnostic-only output can be cleared, and absent report paths do not enable report access. This matches coverage's path-based button policy; interactive OS folder opening and external report deletion were not exercised by the DOM tests.

Verification: `npm --prefix frontend test`, `npm --prefix frontend run build`, `go test -race -count=1 ./...`, and `git diff --check` passed. npm audit reports six vulnerabilities in the existing Vite toolchain dependencies (one moderate, five high); dependency-toolchain remediation is outside this button fix.

### R5 — Resolved: reserve report paths with exclusive creation

- [x] Replace `Stat` followed by `Create` with atomic exclusive file creation shared by both report types.
- [x] Retry the numeric suffix only for `os.ErrExist`; return other errors.
- [x] Add concurrent-writer tests that verify distinct paths and preservation of existing contents.

Location: `internal/processing/services/duplicate_report.go:19–60` at the reviewed head; coverage uses the same path-selection helper.

Scope clarification: the application's execution gate already prevents parallel runs within one application instance. This is defensive hardening against external writers or separate processes, not a supported parallel-execution workflow.

At the reviewed head, separate writers could select the same free path and `os.Create` could truncate an existing report. The helper also treated every `Stat` error as evidence that a path was available. The sequential rapid-run test did not cover atomic reservation.

Implementation: both report writers now use `os.OpenFile` with `os.O_WRONLY|os.O_CREATE|os.O_EXCL`. The shared `createReportFile` helper in `internal/processing/services/report_file.go` attempts creation directly, increments the existing numeric suffix only on `os.ErrExist`, and returns other errors immediately. Timestamp formatting, permissions, report contents, and incomplete-file cleanup behavior remain unchanged. No locks or parallel application execution were added.

Regression coverage: `internal/processing/services/report_file_test.go` simulates a file appearing just before creation for both report types, verifies retry naming and preservation of the original content, and checks permission/missing-directory errors are returned without retries. A service-level concurrency test checks distinct paths and matching contents for 16 simultaneous report writers of each type. The deterministic collision tests failed before the change and pass afterward; existing report write/close/cleanup tests still pass.

Acceptance verified: exclusive creation prevents overwriting existing reports, and non-collision filesystem errors return without suffix retries.

Verification: targeted report-file tests passed 50 repetitions under `-race`; `go test -race -count=1 ./...`, `go build ./...`, `go vet ./...`, and `git diff --check` passed.

### R6 — Medium: `DiagnosticSink.Submit` and `Close` have inconsistent admission semantics

- [ ] Choose and document the sink's producer/closure concurrency contract.
- [ ] Prefer one mutex for closed-state admission, retention, and enqueueing unless measurement justifies a more complex design.
- [ ] Add concurrent submission/closure tests.

Location: `internal/processing/services/diagnostics.go:68–103` at the reviewed head.

A submission can pass the initial closed check, be delayed until `Close` returns, append a retained diagnostic, and then return false at the second closed check without queueing the log event. Thus a rejected submission can alter a post-close snapshot. Current flows generally join producers first, but the API does not clearly state that restriction.

Alternatively, explicitly require all producers to finish before `Close` and simplify the implementation around that contract.

Acceptance: the documented contract is unambiguous; rejected submissions do not appear in retained diagnostics, and closure/draining behavior is covered by tests.

### R7 — Medium: CI does not execute all test packages

- [ ] Run recursive Go package tests so service-package and nested validation tests execute in CI.
- [ ] Include race, vet, and normal build checks without introducing an elaborate pipeline.
- [ ] Run frontend behavioral tests (`npm --prefix frontend test`) after installing development dependencies.

Location: `.github/workflows/ci.yml:29,47` at the reviewed head.

CI currently runs only `./tests/unit_tests` and `./tests/e2e_tests`. It skips the new tests under `internal/processing/services` and nested validation tests.

Suggested checks:

```sh
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build ./...
```

Acceptance: every test package is exercised and the normal build is checked by CI. The simulator-specific regression in R1 was resolved by removing that configuration.

### S1 — Cleanup: remove obsolete code and duplicate cleanup

- [ ] Remove unused `drainHashErrors` from `internal/processing/flows/find_missing_content.go`.
- [ ] Remove the duplicate `defer memory.CloseAndWait()` in that flow.

Keep the sink-based path as the sole error-handling implementation.

### S2 — Consistency: use one completed diagnostic source of truth

- [ ] Evaluate removing separate completed diagnostic publication when diagnostics are already part of the concrete result.
- [ ] Ensure publisher inputs cannot silently disagree about completed diagnostics.

The publisher currently accepts result diagnostics and a separate diagnostic snapshot, then overwrites result diagnostics in `PublishTo`. Prefer deriving completed diagnostics from the concrete result where possible; preserve the standalone running/cancelled/inconclusive application state.

### S3 — Correctness: defensive copies must include nested duplicate slices

- [ ] Copy nested `FileHash.DuplicatesFound` slices at publication/getter boundaries.
- [ ] Add a mutation-isolation test for nested duplicate results.

Location: `internal/processing/runner.go`, `copyDuplicateResult` and `PublishDuplicateResult`.

Copying only the outer `Groups` slice leaves nested duplicates aliased to stored state. Mutating a result returned by a getter must not alter later getter results.

### S4 — Simplicity: avoid speculative classification layers

- [ ] Reassess the handler abstraction only if simplification is warranted; do not add more classification layers without concrete production policies.

The production handler always returns `ErrorDiagnostic`. Keeping that policy is currently accepted. This is a design consideration, not a requirement to invent new classifications or remove an otherwise justified callback.

### S5 — Measurement: asynchronous logging memory trade-off

- [ ] Add a high-volume failure test or benchmark measuring queue size, peak memory, and logger throughput.
- [ ] Revisit asynchronous logging only with evidence; synchronous logging may be simpler if adequate.

The unbounded log queue retains original errors in addition to retained diagnostics. This remains an acknowledged trade-off, not an automatic merge blocker. Do not introduce dropped diagnostics or worker-blocking behavior without explicitly reconsidering the lossless requirement.

### S6 — Test isolation: E2E cleanup shares a directory across test processes

- [ ] Give each E2E test its own `t.TempDir()` and scope cleanup to that directory.

Separate concurrent test commands interfered through shared `./test_files/` cleanup. Isolated reruns passed. This is a test infrastructure limitation observed during review, not evidence of a production data race.

## Review Verification Baseline

At reviewed head `21da4b8`:

| Check | Result |
| --- | --- |
| `go test ./...` | Passed |
| `go test -race -count=1 ./...` | Passed in isolation |
| `go vet ./...` | Passed |
| `cd frontend && npm run build` | Passed |
| `go build -tags debug_progress ./...` | Failed; see R1 |
| Internal statement coverage using `-coverpkg=./internal/...` | 78.5% on an isolated run |
| Targeted discovery cancellation reproduction | Blocked callback reproduced; see R3 |

Coverage command:

```sh
go test -count=1 -coverpkg=./internal/... -coverprofile=/tmp/dude-cover.out ./...
go tool cover -func=/tmp/dude-cover.out
```

Do not run separate E2E-bearing test commands concurrently until S6 is resolved. Wails desktop build and interactive UI behavior were not verified during this review; the frontend production build is not a substitute for behavioral UI tests.

After addressing the checklist, rerun all checks, update this section with current results, and mark only verified items complete.
