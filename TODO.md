## Planned Features

1. [x] Target Coverage Check — Find Missing Content verifies source content against a target directory using the shared matching pipeline. See [Usage](README.md#usage); the historical target-coverage plan was removed.

## Later

- Upgrade Wails after confirming a version compatible with the project's next Go toolchain; update the CLI, Go module, generated bindings, VS Code tasks, and release build verification together.
- Remediate the existing frontend dependency audit findings with a compatible Vite/toolchain upgrade; recheck frontend tests, Wails asset builds, and supported Node versions.
- Remove unused code.
- Make `ExecutionParams` immutable after separating raw requests from resolved execution configuration.
- Review log levels and details.
- Measure diagnostic logging queue size, peak memory, and throughput under high-volume failures. Revisit the unbounded asynchronous queue only with evidence; preserve lossless diagnostic retention unless the requirements explicitly change.
- Isolate E2E test data with per-test `t.TempDir()` ownership and scoped cleanup. Separate test processes currently interfere through shared `./test_files/` cleanup; verify concurrent test commands no longer delete one another's fixtures.
- Test with large files and constrained buffers.
- Throttle or summarize repeated filesystem errors before sending them to the UI.
- Simplify frontend state flow, including mode-independent diagnostic naming/state and tests for mode switching, new runs, and Full Reset.
- Validate macOS permissions, packaging, signing, notarization, and runtime behavior on real hardware.
- Consider a duplicate-file deletion feature.

## Done

- complete the unified diagnostic pipeline and agreed PR #10 review fixes; see `DIAGNOSTIC_SYNC_PLAN.md`
- publish terminal application state before frontend completion/error/cancellation notifications
- make discovery progress sends cancellation-aware and join hashing progress trackers on error paths
- reserve duplicate/coverage report paths with exclusive creation and collision-only suffix retries
- use one sink mutex and derive completed diagnostics from the concrete result
- deep-copy nested duplicate results at collection, publication, and getter boundaries
- run every Go test package with race detection and frontend behavioral tests/build in CI
- remove the unused progress simulator while preserving normal debug builds and logging
- prevent concurrent executions and cancellation/restart overlap
- handle cache initialization, reads, writes, and connection cleanup without panics
- report directory-walk permission and filesystem errors without silently skipping files
- fix Paranoid Mode correctness, worker completion, and comparison error handling
- handle file read errors during hashing
- distinguish missing cache records from database errors
- add SQL repository and database lifecycle tests
- collect file metadata during directory walking
- skip uniquely sized files before hashing
- test same-size files with different contents
- add production hashing benchmarks and comparison workflow
- allow users to change result groups per page dynamically
- detect CSV flush and close errors and remove incomplete result files
- validate writable directories without modifying existing user files
- complete progress phases explicitly, including zero-work and all-failed phases
- close and await every execution-owned channel and worker on all exit paths
- synchronize Full Reset with active execution cleanup and shared state
- report processing failures clearly in the UI
- correct release metadata and document cache, logs, CSV output, dependencies, and supported platforms
  
## Notes

  1. merge time and size and the rest to a single blob and work with the blob after?
  1. use must pattern?
