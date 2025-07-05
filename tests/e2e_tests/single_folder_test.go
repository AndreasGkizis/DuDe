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

func Test_SingleFolder_WithDuplicates2(t *testing.T) {
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
