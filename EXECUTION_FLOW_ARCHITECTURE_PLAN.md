# Reusable Execution-Flow Architecture

**Status:** In Progress  
**Decision date:** 2026-08-29  
**Current step:** 6

## Purpose and Scope

This document is the authoritative implementation tracker for restructuring DuDe into reusable execution flows. It covers the execution architecture, shared processing boundaries, lifecycle ownership, and the review sequence for that restructuring.

Coverage-specific matching rules, reporting, UI behavior, failure semantics, and end-to-end scenarios remain authoritative in [TARGET_COVERAGE_PLAN.md](TARGET_COVERAGE_PLAN.md). This tracker must not be used to weaken or replace that product contract.

The architecture work must preserve duplicate detection as a usable operation after every completed step. Coverage execution remains disabled until the reusable architecture, coverage backend flow, Wails-facing result API, and frontend adapter are connected.

## Verified Baseline

Verified on 2026-08-29 before further production-code changes:

- Repository toolchain: Go 1.24.13 through `mise.toml`.
- Complete unit suite: passing.
- Complete end-to-end suite: passing.
- Baseline command:

  ```bash
  GOCACHE=/tmp/dude-go-build mise exec -- go test -count=1 ./tests/unit_tests ./tests/e2e_tests
  ```

Existing uncommitted Step 1 work is part of the Step 1 review boundary and must be preserved.

## Target Architecture

The execution path will be:

```text
FrontendApp lifecycle → flow registry → selected ExecutionFlow → shared processing services → typed result publication
```

Each execution flow owns:

- Its `ExecutionMode`.
- Validation of mode-specific arguments.
- Orchestration of reusable processing stages.
- Its typed result.

`FrontendApp` continues to own:

- `ExecutionGate`, cancellation, and Full Reset.
- Shared argument resolution and validation.
- Error reporting through `Reporter.ReportError`.
- Publication of successful results under `stateMu`.
- Wails-facing result APIs.

The internal `ExecutionFlow` contract exposes mode, validation, and execution operations. A registry keyed by `ExecutionMode` selects the flow and prevents mode switches from accumulating in `FrontendApp`. Every execution request must provide an explicit recognized mode. Empty and unknown modes are rejected; the backend never infers a mode from the supplied folders or other arguments.

Duplicate and missing-content data stay structurally separate in concrete result types. No shared result interface or central result envelope is introduced at this stage. Cancellation and failure produce no result.

## Review and Completion Rules

1. Complete, review, and commit each numbered step independently.
2. Run the step's focused tests plus every verification explicitly required by that step.
3. Mark a checkbox complete only after its review boundary passes and the step is committed.
4. Preserve existing duplicate behavior, lifecycle guarantees, CSV output, progress behavior, and tests throughout the restructuring.
5. Do not connect coverage execution early; its Start button remains disabled until the backend flow and result API are connected.

## Implementation Steps

- [x] **1. Finish execution data contracts.**
  - Retain `ExecutionModeDuplicates` and `ExecutionModeFindMissingContent` and require callers to provide one explicitly.
  - Do not default or infer the execution mode when it is empty.
  - Retain the source and target find-missing-content folder fields.
  - Use separate concrete result types without a speculative shared result abstraction.
  - Derive missing-content presentation from result data; keep cancellation and inconclusive failure in the execution lifecycle rather than the result model.
  - Keep `DuplicateResult` and `MissingContentResult` structurally separate.
  - Regenerate Wails bindings under Go 1.24.
  - Review boundary: execution models, JSON/Wails contracts, generated bindings, and focused model coverage only; no flow routing or coverage execution.
  - Verification: focused model tests, the complete Go unit/E2E suites, frontend build, and forced Wails build.

- [x] **2. Separate shared resolution from flow validation.**
  - Keep cache/results fallback, worker-count resolution, buffer-size resolution, and writable-results validation shared.
  - Move directory requirements behind mode-specific validation.
  - Reject empty and unknown execution modes explicitly while preserving existing validation messages where applicable.
  - Add Find Missing Content validation for exactly one readable source directory and exactly one readable target directory.
  - Review boundary: argument resolution and validation only; execution still uses duplicate detection and Find Missing Content execution remains disabled.
  - Verification: focused argument-resolution, shared-validation, and flow-validation tests followed by the complete unit suite.

- [x] **3. Introduce the flow contract and registry.**
  - Keep execution flows together under `internal/processing/flows`.
  - Add an internal `ExecutionFlow` interface with mode, validation, and run operations.
  - Add a registry keyed by `ExecutionMode`.
  - Resolve only explicit, recognized modes; do not provide a registry fallback.
  - Register duplicate mode first; do not connect coverage execution yet.
  - Review boundary: flow selection and registration only; duplicate orchestration may still be adapted internally and coverage remains unregistered.
  - Verification: focused contract, registry, empty-mode, and unknown-mode tests followed by the complete unit suite.

- [x] **4. Establish importable processing package boundaries.**
  - Move reusable processing primitives out of the root `processing` package into `internal/processing/services`, which `processing/flows` can import without creating an import cycle.
  - Establish clear homes for discovery, candidate hashing and cache access, equivalence confirmation, and result writing before moving orchestration into a flow.
  - Keep the existing duplicate runner as a temporary adapter while files and dependencies move.
  - Preserve behavior and avoid redesigning the algorithms during this package-only step.
  - Keep compatibility wrappers only where they are necessary to preserve the existing public processing API and tests during the move.
  - Review boundary: package ownership and dependency direction only; duplicate orchestration remains temporarily in the root runner and Find Missing Content remains unregistered.
  - Verification: focused package tests and compile checks followed by the complete unit suite.

- [x] **5. Adapt duplicate detection into the first flow.**
  - Move the current duplicate orchestration behind the flow contract.
  - Return a typed duplicate result instead of mutating application result state during processing.
  - Preserve discovery, progress phases, cache fallback, CSV format, cancellation, and Paranoid Mode behavior.
  - Require all existing duplicate unit and end-to-end tests to remain unchanged and passing.
  - Review boundary: duplicate-flow adaptation only; do not add coverage matching behavior.
  - Verification: focused duplicate-flow tests followed by the unchanged complete unit/E2E suites.

- [ ] **6. Make `FrontendApp` the mode-independent lifecycle shell.**
  - Resolve the selected flow, resolve and validate shared execution arguments, run flow-specific validation, acquire the existing gate, and invoke the flow.
  - Publish results only after successful completion.
  - Clear stale duplicate and coverage state when a run starts.
  - Preserve cancellation as a normal stopped state and route processing failures through `Reporter.ReportError`.
  - Keep `ExecutionGate`, cancellation, Full Reset, shared argument resolution, error reporting, synchronized publication, and Wails-facing APIs in `FrontendApp`.
  - Review boundary: application lifecycle and result publication only; coverage execution remains disabled.
  - Verification: focused lifecycle, state-clearing, cancellation, error-reporting, concurrent-start, and Full Reset tests, including focused race coverage.

- [ ] **7. Extract shared discovery.**
  - Provide one cancellable discovery component usable by multiple flows.
  - Preserve current handling of hidden files, nested directories, symlinks, special entries, metadata failures, and traversal errors.
  - Keep independently discovered collections separate.
  - Review boundary: traversal and file collection only; hashing and equivalence behavior remain unchanged.
  - Verification: focused discovery parity, traversal-error, and cancellation tests followed by duplicate regression tests.

- [ ] **8. Extract shared candidate hashing and cache access.**
  - Reuse size-based candidate selection, MD5 calculation, cache validation, worker limits, buffers, and cancellation.
  - Allow callers to choose candidates so coverage can avoid hashing irrelevant target sizes.
  - Preserve cache-failure fallback and duplicate results.
  - Review boundary: candidate selection, hashing, and cache mechanics only; do not implement directional matching.
  - Verification: focused size-filtering, hashing, cache, fallback, worker-limit, buffer, and cancellation tests followed by duplicate regression tests.

- [ ] **9. Extract shared equivalence confirmation.**
  - Centralize size/hash equivalence and optional byte-for-byte Paranoid confirmation.
  - Support injected hashing in tests for deterministic collision scenarios.
  - Preserve duplicate comparison behavior, failures, and cancellation.
  - Review boundary: equivalence semantics only; source-to-target aggregation remains coverage-specific work.
  - Verification: focused ordinary-match, Paranoid Mode, injected-collision, comparison-error, and cancellation tests followed by duplicate regression tests.

- [ ] **10. Complete the architecture checkpoint.**
  - Run the complete unit/E2E suites and race suites, the frontend build, and the forced Wails build.
  - Update the corresponding architecture-related checkboxes in `TARGET_COVERAGE_PLAN.md` to reflect only work that has actually passed its review boundary.
  - Document the short recipe for adding another flow: define its models, implement the interface, register it, expose its typed result, and add its frontend adapter.
  - Begin the coverage matcher only after this checkpoint passes.
  - Review boundary: regression verification, architecture documentation, and tracker synchronization only; no coverage matcher implementation.
  - Verification: every command in the Verification section plus review of the architecture-related coverage-plan checkboxes.

## Verification

Use Go 1.24 through the repository's mise configuration. Each step also runs its focused tests before the applicable commands below.

```bash
GOCACHE=/tmp/dude-go-build mise exec -- go test -count=1 ./tests/unit_tests ./tests/e2e_tests
GOCACHE=/tmp/dude-go-build mise exec -- go test -race -count=1 ./tests/unit_tests ./tests/e2e_tests
cd frontend
npm ci --include=dev
npm run build
cd ..
GOCACHE=/tmp/dude-go-build mise exec -- wails build -f
```

If Wails binding generation is run separately, it must also use the repository's Go 1.24 mise environment. Record sandbox or platform limitations honestly; a restricted-environment failure is not a project regression unless reproduced in a supported environment.

## Relationship to Target Coverage

This tracker owns the reusable architecture work through the architecture checkpoint. After Step 10 passes, coverage implementation continues under [TARGET_COVERAGE_PLAN.md](TARGET_COVERAGE_PLAN.md), including directional matching, outcome rules, CSV reporting, frontend behavior, and coverage-specific unit and end-to-end scenarios.

The two trackers must remain synchronized, but completion in one does not imply completion in the other. In particular, coverage's Start button remains disabled until the coverage flow, lifecycle routing, Wails-facing result API, and frontend adapter are connected and verified.
