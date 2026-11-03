# Development

## Requirements

- Go 1.24+
- Node.js 18+
- Wails v2.11
- GTK3 and WebKitGTK 4.1 on Linux

Install Wails:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

The `webkit2_41` Linux build tag is configured in `wails.json`.

## Run In Development

```bash
wails dev
```

## Debug Builds

VS Code and Zed debug tasks build the normal application with debugging symbols. Debug builds run real scans; there is no separate simulated progress mode.

```bash
wails build -debug
```

## Build The Application

```bash
wails build
```

Build output is written to `build/bin`.

Supported release targets are Linux AMD64 and Windows AMD64. macOS validation and packaging remain future work.

To validate only the frontend:

```bash
npm --prefix frontend run build
```

## Tests

```bash
go test ./tests/unit_tests
go test ./tests/e2e_tests
```

## Benchmarks

```bash
go test ./tests/benchmarks -run '^$' \
  -bench '^BenchmarkHashAndGroupPipeline$' \
  -benchmem -benchtime=1s -count=10
```

See `tests/benchmarks/README.md` for before-and-after comparisons.

## Dead Code

Install the Go dead-code analyzer:

```bash
go install golang.org/x/tools/cmd/deadcode@latest
```

Run it against production code or include test-only helpers:

```bash
deadcode ./...
deadcode -test ./...
```
