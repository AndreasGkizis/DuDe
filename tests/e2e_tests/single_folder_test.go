package e2e_tests

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

// All your TestDuplicateScanner_* functions remain largely the same,
// as they already execute the built binary and don't directly import your main package.
func Test_SingleFolder_NoDuplicates(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

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

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run() // Run the CLI app
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	_, errFinal := readResultsFile(t, tempbinDir)

	if errFinal == nil {
		t.Errorf("Expected no results file to be produced, but file exists.")
	} else if !os.IsNotExist(errFinal) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
	}
}

func Test_SingleFolder_WithDuplicates(t *testing.T) {
	var stderr bytes.Buffer

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

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{
		"fileA.txt",
		"fileC.txt",
	}
	csvLines, err := readResultsFile(t, tempbinDir)

	if err != nil {
		t.Error("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

func Test_SingleFolder_EmptyFolder(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)
	tempDir, cleanup := createTestFiles(t, map[string]string{})
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	_, err = readResultsFile(t, tempbinDir)
	if err == nil {
		t.Error("Expected no results file for empty folder, but found one")
	}
}

func Test_SingleFolder_HiddenFiles(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	files := map[string]string{
		"file1.txt":         "duplicate content",
		".hidden/file2.txt": "duplicate content",
		"file3.txt":         "unique content",
	}

	tempDir, cleanup := createTestFiles(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{"file1.txt", "file2.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

func Test_SingleFolder_SpecialCharacters(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	files := map[string]string{
		"file with spaces.txt":        "duplicate content",
		"file-with-special-!@#$%.txt": "duplicate content",
		"normal_file.txt":             "unique content",
	}

	tempDir, cleanup := createTestFiles(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{"file with spaces.txt", "file-with-special-!@#$.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

func Test_SingleFolder_DifferentSizes(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create files with same content but different sizes
	files := map[string]string{
		"small.txt":  "content",
		"large1.txt": "content" + string(make([]byte, 1024)), // 1KB file
		"large2.txt": "content" + string(make([]byte, 1024)), // Same content as large1.txt
	}

	tempDir, cleanup := createTestFiles(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Use filepath.ToSlash to ensure consistent path separators
	cmd := exec.Command(binaryPath, "-s="+tempDir)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{"large1.txt", "large2.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
