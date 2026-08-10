## Before Release

1. Make CSV output atomic and check flush, close, and filename-collision errors.
2. Replace inferred progress completion with an explicit phase lifecycle.
3. Add failure-path tests for file changes, permissions, cache errors, cancellation, symlinks, and output failures.
4. Add pull-request CI for tests, race detection, vet, frontend builds, and platform compilation.
5. Pin release tooling, add checksums, test packages, and sign supported release builds.
6. Clean release metadata and document cache, logs, output, and platform requirements.

## Later

- Remove unused code.
- Review log levels and details.
- Test with large files and constrained buffers.
- Simplify frontend state flow.
- Validate macOS permissions, packaging, signing, notarization, and runtime behavior on real hardware.
- Consider a duplicate-file deletion feature.

## Done
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
  
## Notes
  1. merge time and size and the rest to a single blob and work with the blob after?
  1. use must pattern?
