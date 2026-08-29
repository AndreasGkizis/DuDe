# Target Coverage Using DuDe’s Duplicate Rules

**Status:** Planned  
**Decision date:** 2026-08-18

## Summary

Add **Check Target Coverage** as a directional mode of DuDe’s existing duplicate detection:

- **Find Duplicates:** discover equivalent files anywhere across selected folders.
- **Check Target Coverage:** verify that every file DuDe checks in one source folder has an equivalent file in one target folder.

Coverage must reuse the same discovery, hashing, cache, and optional Paranoid Mode comparison logic as duplicate scanning.

The feature reports findings only. It does not copy, move, rename, or delete files.

## Product Behavior

1. The user selects one source folder and one target folder.
2. DuDe recursively discovers files using the same walker and eligibility rules as Find Duplicates.
3. Every source file is checked for equivalent content somewhere in the target.
4. Filenames and relative folder locations do not need to match.
5. Files that exist only in the target are ignored.
6. One target file may cover multiple byte-identical source files.
7. Every uncovered source path appears separately in the missing-files result.
8. The final result shows:
   - total source files;
   - covered source files;
   - missing source files;
   - a missing-file list when applicable.
9. An empty source produces a no-op warning, not a successful coverage claim.
10. Identical or nested source and target folders are allowed, with a warning that files may match themselves.

## Mode and Folder Controls

1. Add a top-level operation selector with:
   - **Find Duplicates**
   - **Check Target Coverage**
2. Default to **Find Duplicates** to preserve existing behavior.
3. Duplicate mode retains the existing dynamic directory list.
4. Coverage mode shows exactly:
   - **Source Folder**
   - **Target Folder**
5. Switching modes:
   - clears all folder selections;
   - clears displayed results and status;
   - preserves shared advanced settings.
6. Disable mode switching while an execution is active.
7. Full Reset:
   - cancels and waits for active processing using the existing execution gate;
   - clears duplicate and coverage results;
   - clears both modes’ folder selections;
   - returns the UI to Find Duplicates mode;
   - preserves the existing cache-reset contract.
8. Detect identical or nested source/target selections and display a non-blocking warning explaining that coverage may be satisfied by the same physical files.

## Shared File-Discovery and Equivalence Engine

1. Do not build an independent approximation of duplicate detection for coverage mode.
2. Extract or reuse one shared discovery and equivalence pipeline for both modes.
3. Preserve the current walker’s filesystem-entry behavior exactly, including its existing treatment of hidden files, symbolic links, and other non-directory entries.
4. Avoid changing which files Find Duplicates currently considers as part of this feature.
5. Preserve the current duplicate-equivalence sequence:
   - discover file metadata;
   - use file size to identify possible matches;
   - calculate or retrieve the cached MD5 hash;
   - group or match files by hash;
   - perform byte-for-byte confirmation when Paranoid Mode is enabled.
6. Coverage must keep source and target collections logically separate.
7. A duplicate found only within the source folder must never count as target coverage.
8. A duplicate found only within the target folder is irrelevant unless it matches source content.
9. Without Paranoid Mode, matching size and MD5 means equivalent, exactly as in Find Duplicates.
10. With Paranoid Mode, at least one target candidate must pass byte-for-byte comparison before the source file is covered.
11. Continue using the existing cache validation based on file path, size, and modification time.
12. Preserve the configured worker count, buffer size, cancellation behavior, progress reporting, and cache-failure fallback behavior.

## Coverage Algorithm

1. Walk the source and target trees using the shared discovery implementation.
2. Keep separate source and target file collections even when paths overlap.
3. Count every discovered source file individually.
4. Build an index of target files by file size.
5. For each source file:
   - if the target has no file of the same size, mark the source file missing;
   - otherwise, calculate or retrieve the source hash;
   - calculate or retrieve hashes for relevant target candidates;
   - find target candidates with the same hash;
   - when Paranoid Mode is disabled, accept a hash match;
   - when Paranoid Mode is enabled, byte-compare candidates until a confirmed match is found.
6. One confirmed target candidate may cover any number of source files with equivalent content.
7. Do not consume target candidates or require matching source/target copy counts.
8. Ignore target files whose known sizes cannot match any source file.
9. Sort missing files by full source path before storing, displaying, or reporting them.
10. Check cancellation throughout traversal, hashing, comparison, collection, and report writing.
11. Do not publish a successful or partial result after cancellation.

## Accuracy and Failure Handling

1. Coverage should produce one of these outcomes:
   - **Complete:** every source file is covered;
   - **Missing:** one or more source files are not covered;
   - **Empty Source:** no source files were available to check;
   - **Inconclusive:** an execution error prevented DuDe from proving coverage.
2. Source traversal or hashing errors make the result inconclusive.
3. Target traversal gaps make the result inconclusive because matching content may exist in the inaccessible subtree.
4. Failure to read or compare a potentially matching target file makes the result inconclusive.
5. An unreadable target-only file may be ignored when its known size proves it cannot match any source file.
6. Do not present partial coverage counts as a trustworthy final result after an inconclusive run.
7. Route processing errors through the existing `Reporter.ReportError` and `errorUpdate` flow.
8. Keep user cancellation as a normal stopped state rather than a processing failure.
9. Clear stale results at the beginning of every execution so failed or cancelled runs cannot display results from a previous run.
10. CSV write, flush, close, or cleanup errors make the execution fail visibly.

## Backend Interfaces and Models

1. Add a typed execution mode, for example:
   - `duplicates`
   - `coverage`
2. Extend `ExecutionParams` with:
   - execution mode;
   - source directory;
   - target directory.
3. Retain the existing `Directories` field for duplicate mode.
4. Treat an omitted or zero-value execution mode as duplicate mode for backward compatibility with existing Go tests and callers.
5. Apply mode-aware validation:
   - duplicate mode requires at least one readable directory;
   - coverage mode requires one readable source directory and one readable target directory;
   - both modes retain shared cache directory, results directory, CPU, and buffer validation.
6. Add a coverage-file model containing:
   - filename;
   - full source path;
   - byte size.
7. Add a coverage-result model containing:
   - outcome;
   - total source-file count;
   - covered source-file count;
   - missing source-file count;
   - ordered missing-file collection;
   - optional generated report path.
8. Add `GetCoverageResult()` for the Wails frontend.
9. Retain `GetResults()` for duplicate groups.
10. Store duplicate and coverage results separately in `FrontendApp`.
11. Protect coverage state with the existing application state mutex.
12. Clear both result types:
   - when a new execution starts;
   - during Full Reset.
13. Route both execution modes through the existing `ExecutionGate`.

## Progress and Status Reporting

1. Reuse the existing progress UI and reporter interface.
2. Use coverage-appropriate phases, such as:
   - Reading Source
   - Reading Target
   - Filtering
   - Hashing
   - Comparing
   - Collecting
   - Reporting
   - Done
3. Ensure every phase completes explicitly, including zero-work phases.
4. Keep progress totals based on actual work rather than the number of folders alone.
5. Display a clear final status for:
   - all files covered;
   - missing files found;
   - empty source;
   - inconclusive failure;
   - user cancellation.
6. Do not label missing files as duplicate groups.
7. Change labels dynamically by mode:
   - duplicate mode retains **Duplicates Found**;
   - coverage mode uses **Source Coverage** or **Missing Files**.

## Coverage Results UI

1. Show a coverage summary containing:
   - `covered / total` source files;
   - missing-file count.
2. When all source files are covered:
   - show a prominent successful state;
   - show zero missing files;
   - do not create or expose a CSV report.
3. When source files are missing:
   - show the result as incomplete coverage rather than an execution error;
   - list every missing source path separately;
   - show filename, full path, and size;
   - provide a **Show** action using the existing file-reveal behavior.
4. When the source is empty:
   - show that no source files were available;
   - do not claim the target is complete;
   - do not create a report.
5. When execution is inconclusive:
   - show the error panel;
   - do not show a successful or missing-files result;
   - do not retain results from the prior execution.
6. Reuse pagination when the missing-file list exceeds the configured page size.
7. Keep duplicate-result rendering unchanged in duplicate mode.
8. Enable **Open Results Folder** only when the current run actually produced a report.

## CSV Reporting

1. Create a CSV only when at least one source file is missing.
2. Name it:
   - `coverage_missing_YYYY_MM_DD_HH_MM_SS.csv`
3. Use these columns:
   - `Filename`
   - `Full Path`
   - `Size (bytes)`
4. Include one row for every missing source path.
5. Sort rows by full source path for deterministic output.
6. Use the existing platform-specific CSV delimiter behavior.
7. Write the existing UTF-8 BOM for spreadsheet compatibility.
8. Save the file in the selected results directory.
9. Produce no coverage CSV for:
   - complete coverage;
   - an empty source;
   - a cancelled execution;
   - an inconclusive execution.
10. If writing, flushing, or closing fails:
    - return an execution error;
    - remove the incomplete output file.
11. Do not change the existing duplicate-results CSV format or filename.

## Unit Tests

1. Validate duplicate mode with an omitted execution mode remains backward compatible.
2. Validate coverage mode requires both source and target folders.
3. Validate shared cache, results-directory, CPU, and buffer resolution.
4. Prove duplicate and coverage modes use the same discovery rules.
5. Prove renamed files with identical content count as covered.
6. Prove files moved into different target subdirectories count as covered.
7. Prove filename equality without content equality does not count.
8. Prove source-only duplicates do not count as target coverage.
9. Prove target-only duplicates do not affect coverage.
10. Prove one target copy covers multiple identical source files.
11. Prove every uncovered identical source path is listed separately.
12. Cover zero-byte files.
13. Cover hidden files and nested directories according to existing discovery behavior.
14. Cover current symbolic-link and special-entry behavior without changing duplicate mode.
15. Cover empty source and empty target folders.
16. Cover identical source and target folders.
17. Cover source nested under target and target nested under source.
18. Test ordinary MD5 matching with Paranoid Mode disabled.
19. Test byte confirmation with Paranoid Mode enabled.
20. Simulate an MD5 collision and prove Paranoid Mode rejects unequal bytes.
21. Test source traversal and hashing errors.
22. Test target traversal gaps.
23. Test relevant target hashing and comparison errors.
24. Test irrelevant target files that cannot match by size.
25. Test cancellation during each expensive phase.
26. Test deterministic missing-file ordering.
27. Test coverage-result state clearing between executions.
28. Test Full Reset clears duplicate and coverage results.
29. Test missing-only CSV generation and exact headers.
30. Test no CSV for complete or empty-source outcomes.
31. Test partial CSV cleanup after write, flush, or close failure.

## End-to-End Tests

1. Complete coverage with renamed and reorganized target files:
   - all source files covered;
   - target extras ignored;
   - no coverage CSV.
2. Partial coverage:
   - correct total and covered counts;
   - every missing source path returned;
   - correctly ordered CSV created.
3. Multiple identical source files with one target copy:
   - every source file counted as covered.
4. Source-internal duplicates with no target copy:
   - every source path reported missing.
5. Empty source:
   - no-op warning;
   - no pass claim;
   - no CSV.
6. Empty target:
   - every source file reported missing.
7. Paranoid Mode:
   - byte-identical candidates accepted;
   - simulated hash collisions rejected.
8. Cancellation:
   - execution stops cleanly;
   - workers and cache writers finish;
   - no stale or partial result is published.
9. Full Reset during coverage:
   - waits for execution cleanup;
   - clears cache and result state;
   - restores the default UI mode.

## Verification Commands

Run the existing suites unchanged:

```bash
GOCACHE=/tmp/dude-go-build go test -count=1 ./tests/unit_tests ./tests/e2e_tests
```

Run concurrency-sensitive regression coverage:

```bash
GOCACHE=/tmp/dude-go-build go test -race -count=1 ./tests/unit_tests ./tests/e2e_tests
```

Build the frontend:

```bash
cd frontend
npm ci --include=dev
npm run build
```

Build the Wails application from the repository root:

```bash
wails build -f
```

Before implementation, the current unit and E2E baseline is passing.

## Acceptance Criteria

1. A source file is covered only by equivalent content in the target collection.
2. Source-only duplicates cannot create false coverage.
3. Coverage and duplicate modes use the same file-discovery and equality rules.
4. Paranoid Mode has identical meaning in both modes.
5. Complete, missing, empty, inconclusive, cancelled, and reset states are visibly distinct.
6. Missing results are deterministic and inspectable in-app and through CSV.
7. Existing duplicate behavior, reports, tests, and lifecycle guarantees remain intact.
8. The complete Go test suite, race test, frontend build, and Wails build pass.

## Implementation Steps

Use this section as the implementation tracker. Each numbered checkbox is intended to be completed, reviewed, and committed independently. Mark a step complete only after its listed verification passes. Keep duplicate mode working at the end of every step.

- [x] **1. Create the coverage UI foundation.**
  - Add the operation selector, Source Folder and Target Folder controls, overlap warning, mode-aware labels, and coverage-summary placeholders.
  - Clear folder selections and displayed state when switching modes while preserving shared advanced settings.
  - Keep coverage execution disabled until the backend contract and result API are connected.
  - Review boundary: frontend structure, styling, and local state only; no coverage result calculation.
  - Verification: `cd frontend && npm run build` and `GOCACHE=/tmp/dude-go-build wails build -f`.

- [ ] **2. Add the execution-mode contract and mode-aware validation.**
  - Add typed `duplicates` and `coverage` execution modes plus source and target directory fields to `ExecutionParams`.
  - Treat an omitted or zero-value mode as duplicate mode.
  - Require `Directories` in duplicate mode and both Source Folder and Target Folder in coverage mode.
  - Preserve shared cache directory, results directory, CPU, and buffer resolution.
  - Review boundary: models and argument validation only; coverage runner routing remains disconnected and the coverage Start button remains disabled.
  - Verification: focused resolver/model tests, then `GOCACHE=/tmp/dude-go-build go test -count=1 ./tests/unit_tests/handlers/validation ./tests/unit_tests`.

- [ ] **3. Add coverage result models and synchronized application state.**
  - Add the coverage outcome, missing-file, and coverage-result models described above.
  - Store duplicate and coverage results separately under `FrontendApp.stateMu`.
  - Add `GetCoverageResult()` while retaining `GetResults()` unchanged.
  - Clear both result types at execution start and during Full Reset.
  - Review boundary: state and Wails-facing data APIs only; no coverage processing yet.
  - Verification: focused state/reset tests and `GOCACHE=/tmp/dude-go-build go test -count=1 ./tests/unit_tests`.

- [ ] **4. Extract shared file discovery without changing duplicate behavior.**
  - Introduce one discovery entry point used by duplicate and coverage modes.
  - Preserve current hidden-file, nested-directory, symbolic-link, and special-entry behavior exactly.
  - Return enough traversal error information for coverage to become inconclusive without changing duplicate-mode eligibility.
  - Keep source and target discoveries as separate collections.
  - Review boundary: traversal and collection only; hashing and matching remain unchanged.
  - Verification: discovery parity and traversal-error tests, then the existing unit and E2E suites.

- [ ] **5. Extract reusable candidate hashing and cache lookup.**
  - Reuse size filtering, cached MD5 validation, worker count, buffer size, cancellation, and cache-failure fallback for caller-selected candidates.
  - Allow coverage to hash only target sizes that could match a source file.
  - Preserve the duplicate runner's current hashing results and progress behavior.
  - Review boundary: hashing and cache mechanics only; no directional coverage decision yet.
  - Verification: cache, hashing, cancellation, and duplicate-regression unit tests.

- [ ] **6. Extract reusable equivalence confirmation.**
  - Provide one equivalence operation that applies size and MD5 matching and optionally performs byte-for-byte Paranoid Mode confirmation.
  - Allow deterministic hash injection in tests so unequal files can simulate an MD5 collision.
  - Preserve existing duplicate Paranoid Mode behavior and comparison-error handling.
  - Review boundary: comparison semantics only; no source-to-target aggregation.
  - Verification: ordinary matching, Paranoid Mode, simulated collision, comparison-error, and cancellation tests.

- [ ] **7. Implement the directional coverage matcher.**
  - Accept already discovered source and target collections and keep them logically separate.
  - Build the target size index and determine covered or missing status for every source path.
  - Allow one target file to cover multiple identical source files without consuming target candidates.
  - Sort missing files by full source path and return Complete, Missing, or Empty Source.
  - Review boundary: deterministic in-memory matching only; no Wails state, CSV, or frontend changes.
  - Verification: focused table-driven tests for renamed, moved, unequal, repeated, zero-byte, empty, identical, and nested-folder scenarios.

- [ ] **8. Implement the coverage execution runner and outcome rules.**
  - Orchestrate source discovery, target discovery, filtering, hashing, comparison, collection, and reporting phases.
  - Reuse `MemoryManager`, configured workers and buffer, reporter events, and cancellation checks.
  - Convert source failures, target traversal gaps, and relevant target read/comparison failures into Inconclusive.
  - Ignore unreadable target-only files only when their known size cannot match any source file.
  - Publish no result after cancellation or inconclusive failure.
  - Review boundary: backend coverage execution and progress only; no CSV or frontend rendering.
  - Verification: runner tests for every outcome, phase completion, relevant/irrelevant failures, cache fallback, and cancellation during expensive phases.

- [ ] **9. Add missing-only coverage CSV reporting.**
  - Write the documented filename, UTF-8 BOM, delimiter, headers, deterministic rows, and result path.
  - Produce a report only for the Missing outcome.
  - Remove incomplete output and return an error after write, flush, close, or cleanup failure.
  - Leave the duplicate CSV format and filename unchanged.
  - Review boundary: coverage report creation only.
  - Verification: exact-output tests plus injected write, flush, close, and cleanup failures.

- [ ] **10. Route coverage through the Wails execution lifecycle.**
  - Select the duplicate or coverage runner from the resolved execution mode.
  - Route both modes through the existing `ExecutionGate`, reporter error flow, cancellation path, and execution-finished signaling.
  - Ensure execution start, failure, cancellation, and Full Reset clear the correct state without publishing stale results.
  - Review boundary: backend routing and lifecycle integration only; the frontend Start button remains disconnected from coverage.
  - Verification: routing, state-clearing, cancellation, concurrent-start, and Full Reset regression tests, including `go test -race` for focused packages.

- [ ] **11. Connect and complete the coverage frontend.**
  - Send mode, source directory, and target directory in `StartExecution` parameters and enable Start in coverage mode.
  - Fetch `GetCoverageResult()` only for coverage executions and keep duplicate rendering unchanged.
  - Render Complete, Missing, Empty Source, Inconclusive, cancellation, and reset as distinct states.
  - Render every missing file with filename, full path, byte size, pagination, and Show action.
  - Enable Open Results Folder only when the current execution produced a report.
  - Review boundary: frontend-to-backend wiring and coverage presentation only.
  - Verification: frontend build, Wails binding generation/build, and manual checks of both modes.

- [ ] **12. Add coverage happy-path end-to-end tests.**
  - Cover complete coverage with renamed/reorganized files, partial coverage and ordered CSV, one target covering multiple sources, source-only duplicates, empty source, and empty target.
  - Assert result counts, ordered missing paths, report presence or absence, and target-extra behavior.
  - Review boundary: deterministic successful and incomplete executions only.
  - Verification: focused coverage E2E tests followed by the unchanged duplicate E2E suite.

- [ ] **13. Add failure and lifecycle end-to-end tests.**
  - Cover Paranoid Mode acceptance and simulated collision rejection.
  - Cover source errors, target traversal gaps, relevant target failures, and inconclusive result suppression.
  - Cover cancellation and Full Reset while coverage is active, including worker/cache cleanup and default-mode restoration.
  - Review boundary: failure, cancellation, and reset behavior only.
  - Verification: focused E2E tests and the full unit/E2E race command.

- [ ] **14. Complete final regression and release verification.**
  - Run every command in Verification Commands, including the full race suite, frontend build, and forced Wails build.
  - Manually verify both operation modes, overlap warnings, result-folder enablement, cancellation, and Full Reset.
  - Confirm no duplicate-mode result, CSV, lifecycle, or UI behavior regressed.
  - Record any platform limitations honestly, update this plan's status, and mark the TODO item complete only after all acceptance criteria pass.
  - Review boundary: verification and documentation only; no new feature scope.

## Continuation Point

Implementation should begin test-first:

1. Add coverage models and mode-aware validation tests.
2. Add deterministic processing tests proving directional matching.
3. Extract the shared discovery/hash/equality components without changing duplicate behavior.
4. Implement the coverage runner and report writer.
5. Add Wails-facing result state and APIs.
6. Implement the mode-specific frontend controls and results.
7. Add end-to-end tests and complete the full verification sequence.

## Persistence Verification

After creating the documentation:

```bash
git diff -- TODO.md TARGET_COVERAGE_PLAN.md
```

Confirm that:

1. `TARGET_COVERAGE_PLAN.md` contains this entire plan.
2. `TODO.md` links to it.
3. No existing TODO material was removed or rewritten.
4. No application code was changed while persisting the plan.
