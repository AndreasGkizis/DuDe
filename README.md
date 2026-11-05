# DuDe

DuDe is a Go/Wails desktop application for finding duplicate files and checking whether source content exists in a target directory.

## Features

- Duplicate detection across multiple directories
- Find Missing Content checks between source and target directories
- Concurrent file scanning and hashing
- File-size filtering to avoid unnecessary hashing
- Optional SQLite hash cache for faster repeated scans
- Optional byte-for-byte duplicate verification
- In-app paginated results, skipped-path diagnostics, and CSV reports
- Linux and Windows builds

## Installation

Download a release for your platform from [GitHub Releases](https://github.com/AndreasGkizis/DuDe/releases).

Linux requires GTK3 and WebKitGTK 4.1.

macOS is not currently a supported release target.

## Build From Source

Requirements:

- Go 1.24+
- Node.js 18+
- Wails v2
- GTK3 and WebKitGTK 4.1 on Linux

```bash
git clone https://github.com/AndreasGkizis/DuDe.git
cd DuDe
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
wails build
```

For development:

```bash
wails dev
```

## Usage

Choose an operation:

- **Duplicates:** select one or more directories. DuDe filters candidates by size, hashes them, and displays matching content groups.
- **Find Missing Content:** select a source and target directory. DuDe lists source files whose content is not found in the target, regardless of filenames or relative paths.

Paranoid Mode adds byte-for-byte verification. Recoverable filesystem failures appear as skipped-path diagnostics. DuDe does not delete files.

## Runtime Files

- Duplicate reports use `duplicate_results_YYYY_MM_DD_HH_MM_SS.csv`; coverage reports use `coverage_report_YYYY_MM_DD_HH_MM_SS.csv`. Filename collisions add a numeric suffix without overwriting existing reports.
- Duplicate reports are created when duplicates or diagnostics exist; coverage reports are created for every completed coverage run.
- The optional cache is stored as `memory.db` in the selected cache directory.
- If those directories are not selected, Linux and Windows use the executable directory.
- Debug logs are written to `logs/` beside the executable only when Debug Logging is enabled.

See [development.md](development.md) for build/test commands and [TODO.md](TODO.md) for the backlog.

## License

[Apache License 2.0](LICENSE)
