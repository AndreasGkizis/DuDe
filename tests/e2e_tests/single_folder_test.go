package e2e_tests

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func Test_SingleFolder_NoDuplicates(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	files := map[string][]byte{
		"file1.txt":          []byte("content A"),
		"sub/file2.txt":      []byte("content B"),
		"sub/sub2/file3.txt": []byte("content C"),
	}
	tempDir, cleanup := createTestFilesByteArray(t, files)
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

	// Define options for test files
	options := DefaultFileOptions()
	options.DuplicateFileCount = 2     // Create 2 files that will have duplicates
	options.UniqueFileCount = 2        // Create 2 files without duplicates
	options.DuplicatesPerFile = 1      // Create 1 duplicate for each duplicate file (2 total identical files)
	options.FileTypes = []FileType{TextFile, ImageFile} // Create text and image files
	options.Prefix = "test-dup"        // Prefix for filenames

	tempDir, cleanup := createTestFiles(t, options)

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

	// We expect to find duplicates in the results file
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	
	// Verify we have results with duplicates
	if len(csvLines) <= 1 { // header row + at least one result row
		t.Error("Expected to find duplicates in results file")
	}
}

func Test_SingleFolder_EmptyFolder(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)
	tempDir, cleanup := createTestFilesByteArray(t, map[string][]byte{})
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

	files := map[string][]byte{
		"file1.txt":         []byte("duplicate content"),
		".hidden/file2.txt": []byte("duplicate content"),
		"file3.txt":         []byte("unique content"),
	}

	tempDir, cleanup := createTestFilesByteArray(t, files)
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

	files := map[string][]byte{
		"file with spaces.txt":        []byte("duplicate content"),
		"file-with-special-!@#$%.txt": []byte("duplicate content"),
		"normal_file.txt":             []byte("unique content"),
	}

	tempDir, cleanup := createTestFilesByteArray(t, files)
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

	expectedFilenames := []string{"file with spaces.txt", "file-with-special-!@#$%.txt"}
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
	files := map[string][]byte{
		"small.txt":  []byte("content"),
		"large1.txt": []byte("content" + string(make([]byte, 1024))), // 1KB file
		"large2.txt": []byte("content" + string(make([]byte, 1024))), // Same content as large1.txt
	}

	tempDir, cleanup := createTestFilesByteArray(t, files)
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

func Test_SingleFolder_MD5_Collision(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create files with same MD5 hash but different content (simulated MD5 collision)
	files := map[string][]byte{
		"large1.txt": {
			0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
			0x2f, 0xca, 0xb5, 0x87, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89,
			0x55, 0xad, 0x34, 0x06, 0x09, 0xf4, 0xb3, 0x02, 0x83, 0xe4, 0x88, 0x83, 0x25, 0x71, 0x41, 0x5a,
			0x08, 0x51, 0x25, 0xe8, 0xf7, 0xcd, 0xc9, 0x9f, 0xd9, 0x1d, 0xbd, 0xf2, 0x80, 0x37, 0x3c, 0x5b,
			0xd8, 0x82, 0x3e, 0x31, 0x56, 0x34, 0x8f, 0x5b, 0xae, 0x6d, 0xac, 0xd4, 0x36, 0xc9, 0x19, 0xc6,
			0xdd, 0x53, 0xe2, 0xb4, 0x87, 0xda, 0x03, 0xfd, 0x02, 0x39, 0x63, 0x06, 0xd2, 0x48, 0xcd, 0xa0,
			0xe9, 0x9f, 0x33, 0x42, 0x0f, 0x57, 0x7e, 0xe8, 0xce, 0x54, 0xb6, 0x70, 0x80, 0xa8, 0x0d, 0x1e,
			0xc6, 0x98, 0x21, 0xbc, 0xb6, 0xa8, 0x83, 0x93, 0x96, 0xf9, 0x65, 0x2b, 0x6f, 0xf7, 0x2a, 0x70,
		},
		"large2.txt": {
			0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
			0x2f, 0xca, 0xb5, 0x07, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89,
			0x55, 0xad, 0x34, 0x06, 0x09, 0xf4, 0xb3, 0x02, 0x83, 0xe4, 0x88, 0x83, 0x25, 0xf1, 0x41, 0x5a,
			0x08, 0x51, 0x25, 0xe8, 0xf7, 0xcd, 0xc9, 0x9f, 0xd9, 0x1d, 0xbd, 0x72, 0x80, 0x37, 0x3c, 0x5b,
			0xd8, 0x82, 0x3e, 0x31, 0x56, 0x34, 0x8f, 0x5b, 0xae, 0x6d, 0xac, 0xd4, 0x36, 0xc9, 0x19, 0xc6,
			0xdd, 0x53, 0xe2, 0x34, 0x87, 0xda, 0x03, 0xfd, 0x02, 0x39, 0x63, 0x06, 0xd2, 0x48, 0xcd, 0xa0,
			0xe9, 0x9f, 0x33, 0x42, 0x0f, 0x57, 0x7e, 0xe8, 0xce, 0x54, 0xb6, 0x70, 0x80, 0x28, 0x0d, 0x1e,
			0xc6, 0x98, 0x21, 0xbc, 0xb6, 0xa8, 0x83, 0x93, 0x96, 0xf9, 0x65, 0xab, 0x6f, 0xf7, 0x2a, 0x70,
		},
	}

	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Use paranoid mode to detect if the files are actually different
	cmd := exec.Command(binaryPath, "-s="+tempDir, "-p")
	cmd.Stderr = &stderr

	err := cmd.Run()
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

// Test_SingleFolder_ParanoidMode tests that paranoid mode correctly identifies true duplicates
func Test_SingleFolder_ParanoidMode(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create files with identical content but different names
	content, err := createRandomContent(4096) // 4KB of identical content
	if err != nil {
		t.Fatalf("Failed to create random content: %v", err)
	}

	files := map[string][]byte{
		"original.dat": content,
		"copy1.dat": content,
		"subfolder/copy2.dat": content,
		"different.dat": append(content[:len(content)-10], make([]byte, 10)...), // Almost identical
	}

	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() {
		cleanup()
		cleanupBin()
	}()

	// Run with paranoid mode enabled
	cmd := exec.Command(binaryPath, "-s="+tempDir, "-p")
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// Should find the exact duplicates but not the almost-identical file
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}

	// Check that we have exactly the right duplicates
	expectedFilenames := []string{"original.dat", "copy1.dat", "copy2.dat"}
	csvContainsExpected(t, csvLines, expectedFilenames)
	
	// Make sure "different.dat" is not in results
	for _, line := range csvLines {
		if len(line) >= 1 && strings.Contains(line[0], "different.dat") {
			t.Errorf("Found 'different.dat' in results when it should not be detected as duplicate")
		}
	}
}
