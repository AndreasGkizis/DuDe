# Dead Code Management

To ensure this project remains **clean and performant**, we utilize the official Go `deadcode` tool. This tool performs a whole-program reachability analysis to identify code that is never executed.

---

## 🔍 What is `deadcode`?

Unlike standard linters, `deadcode` uses a call-graph analysis starting from the `main` package. It identifies functions that can never be reached during execution. This helps us:

* **Reduce Binary Size:** By removing unreachable code.
* **Improve Maintainability:** By eliminating "ghost" functions that no longer serve a purpose.
* **Clean Refactoring:** Safely identifying what can be deleted after a major change.

---

## Setup

Ensure the tool is installed in your environment:

```bash
go install golang.org/x/tools/cmd/deadcode@latest
```

## Usage

1. To run a basic scan of the entire project:
```bash
deadcode ./...
```

2. Including Tests (Required for this Repo)<br>
Since we use xUnit patterns and utility functions in our _test.go files, a standard scan will flag test-only helpers as unused. To include tests in the analysis, run:
```bash
deadcode -test ./...
```

# Building for all the platforms

