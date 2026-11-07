# Backlog

This list is ordered roughly by usefulness and urgency. Items in **Next** are the best candidates for the next development pass; **Later** items are worthwhile but should not block reliability or release work.

## Next

### 1. Release validation

- [ ] Perform a fresh Wails desktop build (`wails build -f`) and interactive smoke test on the supported Linux and Windows targets.
- [ ] Verify packaged assets, startup, directory selection, duplicate scans, Find Missing Content scans, cancellation, reset, reports, cache, and diagnostics.

### 2. Test realistic file sizes and memory limits

- [x] Add an E2E test with moderately large files and a constrained buffer size.
- [ ] Cover cancellation and hashing errors while large files are being read.

### 4. Fix frontend dependency vulnerabilities

- [ ] Upgrade the Vite/frontend toolchain from the current Vite 3 dependency set in a compatible, deliberate change.
- [ ] Re-run the frontend tests and production build, Wails asset embedding, `npm audit`, and supported Node-version checks.

## Important but not urgent

- [ ] Measure diagnostic logging queue length, peak memory, and throughput under high-volume failures. Keep the current lossless unbounded queue unless measurements and requirements justify a policy change.
- [ ] Simplify frontend state flow, especially mode-independent diagnostic naming/state. Add coverage for mode switching, new runs, and Full Reset.
- [ ] Review log levels and log detail for user-facing and debug output.
- [ ] Remove genuinely unused code after confirming it is not part of the Wails/generated/build surface.
- [ ] Decide whether to separate raw requests from resolved execution configuration, then make `ExecutionParams` immutable after resolution. This is an API/design change, not a cleanup-only task.
- [ ] Review the stale inline hashing note about adding a blob suffix for uniqueness (`internal/processing/services/hashing.go`); either define the requirement and implement it or remove the note.

## Toolchain maintenance

- [ ] Upgrade Wails only after confirming compatibility with the next Go toolchain. Update the CLI, Go module, generated bindings, VS Code tasks, and release-build verification together.

## Future features and platform work

- [ ] Validate macOS permissions, packaging, signing, notarization, and runtime behavior on real hardware before calling macOS supported.
- [ ] Consider a duplicate-file deletion feature. This needs a separate safety/design review; DuDe currently reports findings and does not delete files.

## Done

- [x] Target Coverage Check: Find Missing Content verifies source content against the target directory using the shared matching pipeline.
- [x] Isolate E2E fixtures with per-test `t.TempDir()` ownership so concurrent test processes cannot delete one another's data.
- [x] Complete the unified diagnostic pipeline and the agreed PR #10 review fixes.
- [x] Publish terminal application state before frontend completion/error/cancellation notifications.
- [x] Make discovery progress sends cancellation-aware and join hashing progress trackers on error paths.
- [x] Reserve duplicate/coverage report paths with exclusive creation and collision-only suffix retries.
- [x] Use one sink mutex and derive completed diagnostics from the concrete result.
- [x] Deep-copy nested duplicate results at collection, publication, and getter boundaries.
- [x] Run every Go test package with race detection and frontend behavioral tests/build in CI.
- [x] Remove the unused progress simulator while preserving normal debug builds and logging.
- [x] Prevent concurrent executions and cancellation/restart overlap.
- [x] Handle cache initialization, reads, writes, and connection cleanup without panics.
- [x] Report directory-walk permission and filesystem errors without silently skipping files.
- [x] Fix Paranoid Mode correctness, worker completion, and comparison error handling.
- [x] Handle file read errors during hashing.
- [x] Distinguish missing cache records from database errors.
- [x] Add SQL repository and database lifecycle tests.
- [x] Collect file metadata during directory walking.
- [x] Skip uniquely sized files before hashing.
- [x] Test same-size files with different contents.
- [x] Add production hashing benchmarks and comparison workflow.
- [x] Allow users to change result groups per page dynamically.
- [x] Detect CSV flush and close errors and remove incomplete result files.
- [x] Validate writable directories without modifying existing user files.
- [x] Complete progress phases explicitly, including zero-work and all-failed phases.
- [x] Close and await every execution-owned channel and worker on all exit paths.
- [x] Synchronize Full Reset with active execution cleanup and shared state.
- [x] Report processing failures clearly in the UI.
- [x] Correct release metadata and document cache, logs, CSV output, dependencies, and supported platforms.

## Untriaged ideas

- Merge time and size and the rest to a single blob, then work with the blob.
- Use a MUST pattern.
