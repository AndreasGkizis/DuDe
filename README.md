# DuDe (Duplicate Detector)

[![Go](https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

DuDe is a high-performance command-line utility written in Go that helps you find and manage duplicate files across directories. It identifies duplicates by comparing file contents rather than just filenames, ensuring accurate detection of identical files regardless of their names or locations.

## Features

- 🚀 **Fast and efficient** duplicate file detection
- 🔍 **Content-based comparison** for accurate results
- 📊 Generates detailed **CSV reports** of duplicate files
- 💾 **SQLite database** for caching file hashes
- 📈 **Progress tracking** with visual feedback
- 🖥️ **Cross-platform** (Windows, Linux, macOS)
- 🔄 **Dual-folder mode** for comparing two directories
- 🔒 **Paranoid mode** for extra verification
- 📝 **Configurable** through command-line arguments or config file

## Installation

### Prerequisites
- Go 1.16 or higher
- Git (for cloning the repository)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/DuDe.git
cd DuDe

# Build the executable
go build -o dude ./cmd/main.go

# Make the binary executable (Linux/macOS)
chmod +x dude
```

## Usage

### First Run
On the first run, DuDe will create an `arguments.txt` file in the same directory as the executable. You can edit this file to configure your scan settings.

### Basic Usage

```bash
# Scan a single directory for duplicates
./dude -s /path/to/source/directory

# Compare two directories
./dude -s /path/to/source -t /path/to/target

# Specify custom cache and output locations
./dude -s /path/to/source -c /custom/cache.db -r /custom/results.csv

# Enable paranoid mode for extra verification
./dude -s /path/to/source -p true
```

### Command Line Arguments

| Short | Long          | Description                                      | Default Value              |
|-------|---------------|--------------------------------------------------|----------------------------|
| `-s`  | `--source`    | Source directory to scan (required)              | -                          |
| `-t`  | `--target`    | Target directory to compare with source          | -                          |
| `-c`  | `--cache-dir` | Path to cache database file                      | `./memory.db`              |
| `-r`  | `--results`   | Path to save results CSV file                    | `./results.csv`            |
| `-p`  | `--paranoid`  | Enable paranoid mode for extra verification      | `false`                    |

### Configuration File

You can also configure DuDe by editing the `arguments.txt` file that's created on first run. The file contains the following parameters:

```
SOURCE_DIR=[path to source directory]
TARGET_DIR=[path to target directory (optional)]
CACHE_FILE=[path to cache database file]
RESULT_FILE=[path to results CSV file]
PARANOID_MODE=[true/false]
```

## How It Works

1. **File Indexing**: DuDe scans the specified directories and creates an index of all files.
2. **Hashing**: Each file's content is hashed using a fast hashing algorithm.
3. **Comparison**: Files with matching hashes are compared byte-by-byte (in paranoid mode) to confirm they are identical.
4. **Reporting**: Results are saved to a CSV file for further analysis.

## Acknowledgements

- Built with ❤️ using Go
- Uses modernc.org/sqlite for database operations
- Inspired by the need for a fast, reliable duplicate file finder

## Support

For support, please [open an issue](https://github.com/yourusername/DuDe/issues) on GitHub.

---

Happy duplicate hunting! 🎯
