## Before Release

1. Centralize execution cleanup, synchronize reset state, close background workers, and report execution errors clearly.
2. Correct release metadata and document cache, logs, CSV output, dependencies, and supported platforms.

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
  - detect CSV flush and close errors and remove incomplete result files
  - validate writable directories without modifying existing user files
  - complete progress phases explicitly, including zero-work and all-failed phases
  
## Notes
  1. merge time and size and the rest to a single blob and work with the blob after?
  1. use must pattern?
