package e2e_tests

import (
	"DuDe/internal/common"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// buildBinary builds the Go application binary to be tested.
// It returns the binary path, temp directory, cleanup func.
func buildBinary(t *testing.T) (string, string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp(".", "dude-test-bin-")
	if err != nil {
		t.Fatalf("failed to create temp dir for binary: %v", err)
	}

	binaryName := "dude"
	if os.Getenv("GOOS") == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)

	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/main.go")
	//^^ hacky path, any better option?
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary from ./cmd/dude: %v", err)
	}

	return binaryPath, tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// createTestFiles creates a temporary directory and populates it with the
// specified files and their content for testing.
// It returns the path to the created temporary directory and a cleanup
// function to remove the directory and its contents.
func createTestFiles(t *testing.T, files map[string]string) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp(".", "dude-test-data-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			cleanup()
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
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
