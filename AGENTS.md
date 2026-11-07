# Agent instructions

## Project

DuDe is a Go/Wails desktop application for duplicate-file detection and Find Missing Content checks. Read [`docs/agent/project-context.md`](docs/agent/project-context.md) when the task needs deeper architectural or lifecycle context.

## Repository guidance

- Read [`README.md`](README.md) and [`development.md`](development.md) for user-facing behavior and development commands.
- Use [`TODO.md`](TODO.md) as the prioritized backlog. Do not reopen items listed under **Done** unless implementation evidence shows they regressed.
- Keep changes small and focused. Preserve the existing content-matching semantics, optional-cache degradation, serialized execution ownership, state-before-notification ordering, defensive result copies, and lossless diagnostic retention.
- Do not add file deletion behavior, new diagnostic classification policies, or logging overflow/backpressure behavior without an explicit design decision.
- Treat generated Wails bindings and embedded frontend assets as build outputs: update them only when the relevant Wails workflow requires it.

## Validation

For normal changes, run the relevant focused tests. Before finalizing broader Go/frontend changes, use the CI-equivalent checks documented in `development.md`:

```bash
npm --prefix frontend ci --include=dev
npm --prefix frontend test
npm --prefix frontend run build
go test -race -v -count=1 ./...
go vet ./...
go build ./...
```

Build the frontend before Go checks because `main.go` embeds `frontend/dist`. E2E fixtures are test-owned temporary directories and can be run without the old shared cleanup constraint.

Do not run benchmarks unless explicitly requested.
