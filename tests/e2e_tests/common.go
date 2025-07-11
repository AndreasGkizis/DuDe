package e2e_tests

import (
	"DuDe/internal/common"
	process "DuDe/internal/processing"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var baseDir string = "./test_files/"

// buildBinary builds the Go application binary to be tested.
// It returns the binary path, temp directory, cleanup func.
func buildBinary(t *testing.T) (string, string, func()) {
	t.Helper()

	// Get the absolute path to the project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to get project root path: %v", err)
	}

	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	tempDir, err := os.MkdirTemp(baseDir, "dude-test-bin-")
	if err != nil {
		t.Fatalf("failed to create temp dir for binary: %v", err)
	}

	// Ensure tempDir is absolute
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}

	binaryName := "dude"
	if os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)

	// Use absolute path for the main package
	mainPkgPath := filepath.Join(projectRoot, "cmd", "main.go")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, mainPkgPath)
	cmd.Dir = projectRoot // Set working directory to project root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v\nCommand: %s", err, cmd.String())
	}

	// On Windows, we need to ensure the binary has the .exe extension for execution
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try with .exe if not found (for Windows)
		if !strings.HasSuffix(binaryPath, ".exe") {
			binaryPath += ".exe"
		}
	}

	// Verify the binary exists and is executable
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not found at %s: %v", binaryPath, err)
	}

	return binaryPath, tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// createTestFilesByteArray creates a temporary directory and populates it with the
// specified files and their content for testing.
// It returns the path to the created temporary directory and a cleanup
// function to remove the directory and its contents.
func createTestFilesByteArray(t *testing.T, files map[string][]byte) (string, func()) {
	t.Helper()
	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	// Create temp directory under the base directory
	tempDir, err := os.MkdirTemp(baseDir, "dude-test-data-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Ensure the path is in the correct format for the current OS
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}

	for path, content := range files {
		// Clean the path to handle any path separators correctly for the current OS
		path = filepath.Clean(path)
		fullPath := filepath.Join(tempDir, path)

		// Ensure the directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			cleanup()
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Create the file with the specified content
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			cleanup()
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	return tempDir, cleanup

}

// Reads and parses the CSV results file from the specified directory.
// It returns the parsed CSV lines or an error if any operation fails.
func readResultsFile(t *testing.T, dir string) ([][]string, error) {
	t.Helper()
	filePath := filepath.Join(dir, common.ResFilename)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to open results file %q: %w", filePath, err)
	}

	defer file.Close()

	bla := csv.NewReader(file)

	bla.Comma = process.GetDelimiterForOS()
	allCsvLines, err := bla.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data from %q: %w", filePath, err)
	}
	return allCsvLines, nil
}

// csvContainsExpected iterates through each expected word and verifies its existence within any of the CSV lines.
// If an expected word is not found, it reports a test error.
func csvContainsExpected(t *testing.T, allCsvLines [][]string, expectedWords []string) {
	t.Helper()
	for _, expectedWord := range expectedWords {
		found := false
		for _, line := range allCsvLines {
			if slices.Contains(line, expectedWord) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("File does not contain :\n%q", expectedWord)
		}
	}
}
