package e2etests

import (
	"DuDe/internal/common"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func getResultsFile(dir string) (*os.File, error) {
	filename := common.ResFilename
	fullpath := filepath.Join(dir, filename)

	file, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// buildBinary builds the Go CLI application binary for testing.
// It returns the path to the executable.
func buildBinary(t *testing.T) (string, string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp(".", "dude-test-bin")
	if err != nil {
		t.Fatalf("failed to create temp dir for binary: %v", err)
	}

	binaryName := "dude"
	if os.Getenv("GOOS") == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)

	// **** IMPORTANT CHANGE HERE ****
	// We need to tell 'go build' where our main package is located.cd
	// Assuming your main is at 'cmd/dupescanner/main.go' from the project root.
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/main.go") // Adjust path as needed
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary from ./cmd/dude: %v", err)
	}

	return binaryPath, tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// createTestFiles (same as before)
func createTestFiles(t *testing.T, files map[string]string) (string, func()) {
	// ... (no changes needed here)
	t.Helper()
	tempDir, err := os.MkdirTemp(".", "dupescanner-test-data")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// All your TestDuplicateScanner_* functions remain largely the same,
// as they already execute the built binary and don't directly import your main package.
func TestDuplicateScanner_NoDuplicates(t *testing.T) {

	binaryPath, _, cleanupBin := buildBinary(t)

	files := map[string]string{
		"file1.txt":          "content A",
		"sub/file2.txt":      "content B",
		"sub/sub2/file3.txt": "content C",
	}
	tempDir, cleanup := createTestFiles(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run() // Run the CLI app
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedOutput := "No duplicates were found"
	if !strings.Contains(stdout.String(), expectedOutput) {
		t.Errorf("Expected stdout:\n%q\nGot:\n%q", expectedOutput, stdout.String())
	}
	if stderr.Len() > 0 {
		t.Errorf("Expected no stderr output, got:\n%q", stderr.String())
	}
}

func TestDuplicateScanner_WithDuplicates(t *testing.T) {
	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	files := map[string]string{
		"fileA.txt":      "duplicate content",
		"sub/fileB.txt":  "unique content",
		"sub2/fileC.txt": "duplicate content",
		"sub2/fileD.txt": "another unique content",
	}
	tempDir, cleanup := createTestFiles(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{
		"fileA.txt",
		"fileC.txt",
	}

	// resFile, err := getResultsFile(binaryPath)
	filepath := filepath.Join(tempbinDir, common.ResFilename)
	textFromFile, err := os.ReadFile(filepath)
	lines := strings.Split(string(textFromFile), ",")
	a := IsSubset(expectedFilenames, lines)
	fmt.Print(a)
	// output := stdout.String()
	// not working
	for _, expectedLine := range expectedFilenames {
		if !slices.Contains(lines, expectedLine) {
			t.Errorf("File does not contain :\n%q\nGot:\n%q", expectedLine, textFromFile)
		}
	}
	if stderr.Len() > 0 {
		t.Errorf("Expected no stderr output, got:\n%q", stderr.String())
	}
}

func TestDuplicateScanner_InvalidPath(t *testing.T) {
	binaryPath, _, cleanupBin := buildBinary(t)

	invalidPath := filepath.Join(os.TempDir(), "non-existent-dir-12345") // A path that definitely doesn't exist
	defer func() {
		cleanupBin()
	}()
	cmd := exec.Command(binaryPath, "-s="+invalidPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// We expect an error because the directory doesn't exist, and the app should exit with non-zero
	if err == nil {
		t.Fatalf("CLI app was expected to fail for invalid path, but succeeded.")
	}

	// Check exit code. For CLI apps, a non-zero exit code usually indicates an error.
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Errorf("Expected non-zero exit code for invalid path, got 0.")
		}
	} else {
		t.Errorf("Expected exec.ExitError, got %T: %v", err, err)
	}

	expectedErrorOutput := fmt.Sprintf("Error: stat %s: no such file or directory", invalidPath)
	if !strings.Contains(stderr.String(), expectedErrorOutput) {
		t.Errorf("Expected stderr to contain:\n%q\nGot:\n%q", expectedErrorOutput, stderr.String())
	}

	if stdout.Len() > 0 {
		t.Errorf("Expected no stdout output for error, got:\n%q", stdout.String())
	}
}

func TestDuplicateScanner_NoArgs(t *testing.T) {
	binaryPath, _, cleanupBin := buildBinary(t)

	cmd := exec.Command(binaryPath) // No arguments

	defer func() {
		cleanupBin()
	}()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("CLI app was expected to fail without arguments, but succeeded.")
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Errorf("Expected non-zero exit code for no arguments, got 0.")
		}
	} else {
		t.Errorf("Expected exec.ExitError, got %T: %v", err, err)
	}

	expectedStderr := "Usage: dupescanner <directory>\n"
	if stderr.String() != expectedStderr {
		t.Errorf("Expected stderr:\n%q\nGot:\n%q", expectedStderr, stderr.String())
	}
}

// IsSubset checks if all elements of subsetSlice are present in supersetSlice.
// Uses slices.Contains for efficient lookup within the inner loop.
func IsSubset(subsetSlice, supersetSlice []string) bool {
	for _, subItem := range subsetSlice {
		if !slices.Contains(supersetSlice, subItem) {
			return false // Element from subset not found in superset
		}
	}
	return true // All elements found
}
