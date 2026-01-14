# DuDe (Duplicate Detector)

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-red)](https://wails.io/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](#installation)

## 🎯 Overview

**DuDe**(Duplicate-Detection) is a high-performance, cross-platform desktop tool designed to identify duplicate files.<br>
DuDe utilizes **content-based hashing** and an optional **Paranoid Mode** (byte-by-byte verification) to ensure 100% data integrity.

Built with **Go** and **Wails**, DuDe is a portable single file executable which requires nothing to be installed from the users side ( except webview2)

---

## 🚀 Features

* **Performant Backend**: Concurrent file indexing and hashing.
* **Content-Aware**: Identifies duplicates regardless of filename or location.
* **SQLite Caching**: Persistent hash storage using `modernc.org/sqlite` for faster re-runs.
* **CSV Reporting**: Exports results to a CSV file for analysis.
* **Modern GUI**: A clean, responsive interface that stays out of your way.
* **Paranoid Mode**: Optional byte-for-byte verification to eliminate the theoretical risk of hash collisions.


---

## 📦 Installation

### Recommended: Download Pre-compiled Binaries
You do not need Go or Node.js installed to use DuDe. 

1.  Navigate to the [**Releases**](https://github.com/yourusername/DuDe/releases) page.
2.  Download the package for your Operating System:
    * **Windows**: `.exe` (Self-contained, includes WebView2 installer).
    * **Linux**: `.tar.gz` (Compatible with modern GTK-based distros).
    * **macOS**: `.zip` (Universal Binary for Intel and Apple Silicon).
3.  Launch the application.