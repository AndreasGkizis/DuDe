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

Run the same checks as CI, including every Go test package and the frontend behavioral tests:

```bash
npm --prefix frontend ci --include=dev
npm --prefix frontend test
npm --prefix frontend run build
go test -race -v -count=1 ./...
go vet ./...
go build ./...
```

Build the frontend before the Go checks: `main.go` embeds `frontend/dist`, which is absent on a fresh checkout. Frontend behavioral tests use Node.js's built-in test runner and JSDOM, with Vite loading the actual frontend modules.

Run Go test commands sequentially; separate E2E test processes currently share a cleanup directory.

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
