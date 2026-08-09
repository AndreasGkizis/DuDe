## TODO
  - Remove unused code
  - check logs. levels and details

  - test with large files to provoke buffer overflow ( kinda overkill )
  - simplify frontend state flow

  - make file info work for all builds, also test it! 
  
## FIXME
  - Mac build(needs permissions to execute and write results, need a mac user)

## Features
  - to do or not to do ( delete file feature.)

## Done
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
