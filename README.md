# DuDe

DuDe is a desktop application for finding duplicate files across one or more directories. It is built with Go and Wails.

## Features

- Concurrent file scanning and hashing
- File-size filtering to avoid unnecessary hashing
- Optional SQLite hash cache for faster repeated scans
- Optional byte-for-byte duplicate verification
- In-app paginated results and CSV reports
- Linux, Windows, and macOS builds

## Installation

Download a release for your platform from [GitHub Releases](https://github.com/AndreasGkizis/DuDe/releases).

Linux requires GTK3 and WebKitGTK 4.1.

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

Select one or more directories, configure optional settings, and start the scan. DuDe groups possible duplicates by size, hashes the remaining candidates, and displays confirmed hash matches in the application. Paranoid Mode additionally compares matching files byte by byte.

## License

[Apache License 2.0](LICENSE)
