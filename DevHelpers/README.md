# Development helpers

The automated E2E tests create their own fixtures with `t.TempDir()`. These helpers are not required for normal tests or releases. They are only for manual stress testing with larger files.

## Duplicate stress fixtures

`create_stress_duplicate_fixtures.sh` creates identical file pairs in one directory:

```bash
DevHelpers/create_stress_duplicate_fixtures.sh /tmp/dude-duplicate-stress 2 64
```

Arguments:

```text
OUTPUT_DIR [PAIR_COUNT] [SIZE_MB]
```

The example creates two duplicate pairs, with each file 64 MiB. DuDe should report two duplicate groups when scanning `/tmp/dude-duplicate-stress`.

## Find Missing Content stress fixtures

`create_stress_missing_content_fixtures.sh` creates separate source and target directories:

```bash
DevHelpers/create_stress_missing_content_fixtures.sh /tmp/dude-coverage-stress 2 2 64
```

Arguments:

```text
OUTPUT_DIR [COVERED_COUNT] [MISSING_COUNT] [SIZE_MB]
```

The example creates:

- `source/` with four files
- `target/` with renamed copies of two of those files
- Two source files that do not exist in the target

Run DuDe in **Find Missing Content** mode with:

```text
Source: /tmp/dude-coverage-stress/source
Target: /tmp/dude-coverage-stress/target
```

It should report the two `source-missing-*` files, while the renamed target files should cover the two corresponding source files.

## Manual test procedure

1. Start with small fixtures, such as `2 2 16` or `2 16`.
2. Run the appropriate helper.
3. Start DuDe with `wails dev`, or build it with `wails build` and run `build/bin/DuDe`.
4. Select the matching operation and scan the generated directory.
5. If the UI exposes the buffer setting, try a small value such as 4 KiB.
6. Confirm the expected groups or missing files, report, progress, and diagnostics.
7. Increase the file size only after the small test works.

For example, a 4 GiB duplicate test is:

```bash
DevHelpers/create_stress_duplicate_fixtures.sh /tmp/dude-duplicate-stress 2 1024
```

The scripts use `/dev/urandom`, so they do not block waiting for entropy. They print an estimated total size before creating files, but check available disk space yourself before large runs.

## Cleanup

The scripts never delete fixtures automatically. Remove a test directory after each run:

```bash
rm -rf -- /tmp/dude-duplicate-stress
rm -rf -- /tmp/dude-coverage-stress
```

To remove both only when you are finished with both test sets:

```bash
rm -rf -- /tmp/dude-duplicate-stress /tmp/dude-coverage-stress
```

Use explicit directories under `/tmp` rather than broad commands such as `rm -rf test_files` or deleting the repository. The generated files are not tracked by Git when created outside the repository.
