package e2e_tests

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

// Test_DualFolder_NoDuplicates tests the case where there are no duplicate files between two folders
func Test_DualFolder_NoDuplicates(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create two folders with different content
	folder1Files := map[string][]byte{
		"file1.txt":          []byte("content A"),
		"sub/file2.txt":      []byte("content B"),
		"sub/sub2/file3.txt": []byte("content C"),
	}

	folder2Files := map[string][]byte{
		"fileX.txt":          []byte("content D"),
		"sub/fileY.txt":      []byte("content E"),
		"sub/sub2/fileZ.txt": []byte("content F"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// No duplicates expected, so no results file should be produced
	_, errFinal := readResultsFile(t, tempbinDir)

	if errFinal == nil {
		t.Errorf("Expected no results file to be produced, but file exists.")
	} else if !os.IsNotExist(errFinal) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
	}
}

// Test_DualFolder_WithDuplicates tests the case where there are duplicate files between two folders
func Test_DualFolder_WithDuplicates(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create two folders with some duplicate content
	folder1Files := map[string][]byte{
		"fileA.txt":      []byte("duplicate content"),
		"sub/fileB.txt":  []byte("unique content 1"),
		"sub2/fileC.txt": []byte("another duplicate"),
	}

	folder2Files := map[string][]byte{
		"fileX.txt":      []byte("duplicate content"), // Same as fileA.txt in folder1
		"sub/fileY.txt":  []byte("unique content 2"),
		"sub2/fileZ.txt": []byte("another duplicate"), // Same as fileC.txt in folder1
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// Two sets of duplicates are expected
	expectedFilenames := []string{
		"fileA.txt",
		"fileX.txt",
		"fileC.txt",
		"fileZ.txt",
	}
	csvLines, err := readResultsFile(t, tempbinDir)

	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_EmptyFolders tests the case where both folders are empty
func Test_DualFolder_EmptyFolders(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create two empty folders
	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{})
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{})

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// No duplicates expected for empty folders
	_, err = readResultsFile(t, tempbinDir)
	if err == nil {
		t.Error("Expected no results file for empty folders, but found one")
	}
}

// Test_DualFolder_HiddenFiles tests the case with hidden files that have duplicate content
func Test_DualFolder_HiddenFiles(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with hidden files
	folder1Files := map[string][]byte{
		"visible.txt":        []byte("unique content"),
		".hidden/secret.txt": []byte("duplicate content"),
	}

	folder2Files := map[string][]byte{
		"normal.txt":          []byte("other content"),
		".invisible/data.txt": []byte("duplicate content"), // Same as secret.txt in folder1
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{"secret.txt", "data.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_SpecialCharacters tests the case with files having special characters in their names
func Test_DualFolder_SpecialCharacters(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with files having special characters in names
	folder1Files := map[string][]byte{
		"file with spaces.txt": []byte("duplicate content"),
		"normal_file.txt":      []byte("unique content"),
	}

	folder2Files := map[string][]byte{
		"file-with-special-!@#$%.txt": []byte("duplicate content"), // Same content as "file with spaces.txt"
		"regular.txt":                 []byte("different content"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
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

// Test_DualFolder_DifferentSizes tests the case with files having same content but different sizes
func Test_DualFolder_DifferentSizes(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create files with same content prefix but different sizes
	folder1Files := map[string][]byte{
		"small.txt":  []byte("content"),
		"large1.txt": []byte("content" + string(make([]byte, 1024))), // 1KB file
	}

	folder2Files := map[string][]byte{
		"tiny.txt":   []byte("content"),                              // Same as small.txt
		"large2.txt": []byte("content" + string(make([]byte, 1024))), // Same as large1.txt
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{"small.txt", "tiny.txt", "large1.txt", "large2.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_OneEmptyFolder tests the edge case where one folder is empty
func Test_DualFolder_OneEmptyFolder(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create one folder with files and one empty folder
	folder1Files := map[string][]byte{
		"file1.txt": []byte("content A"),
		"file2.txt": []byte("content B"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{}) // Empty folder

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// No duplicates expected when one folder is empty
	_, err = readResultsFile(t, tempbinDir)
	if err == nil {
		t.Error("Expected no results file when one folder is empty, but found one")
	}
}

// Test_DualFolder_NestedStructure tests the edge case with complex nested directory structures
func Test_DualFolder_NestedStructure(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with complex nested structures
	folder1Files := map[string][]byte{
		"level1/file.txt":                    []byte("duplicate in deep structure"),
		"level1/level2/level3/deep_file.txt": []byte("another duplicate"),
		"level1/level2/unique.txt":           []byte("unique content"),
	}

	folder2Files := map[string][]byte{
		"different/structure/file.txt":         []byte("duplicate in deep structure"), // Same as level1/file.txt
		"totally/different/path/deep_file.txt": []byte("another duplicate"),           // Same as level1/level2/level3/deep_file.txt
		"some/other/file.txt":                  []byte("different content"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	expectedFilenames := []string{
		"file.txt",
		"file.txt",
		"deep_file.txt",
		"deep_file.txt",
	}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_SameFilesButDifferentContent tests the edge case with files having the same names but different content
func Test_DualFolder_SameFilesButDifferentContent(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with same filenames but different content
	folder1Files := map[string][]byte{
		"same_name.txt":      []byte("content version A"),
		"also_same_name.txt": []byte("truly unique content"),
		"another_file.txt":   []byte("duplicate content"),
	}

	folder2Files := map[string][]byte{
		"same_name.txt":      []byte("content version B"), // Same name, different content
		"also_same_name.txt": []byte("different content"), // Same name, different content
		"different_file.txt": []byte("duplicate content"), // Different name, same content as another_file.txt
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// Only the files with same content should be reported as duplicates
	expectedFilenames := []string{"another_file.txt", "different_file.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_CaseSensitivity tests the edge case with files having same names but different case
func Test_DualFolder_CaseSensitivity(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with same filenames but different case
	folder1Files := map[string][]byte{
		"case_test.txt":  []byte("same content"),
		"mixed_CASE.txt": []byte("also same content"),
	}

	folder2Files := map[string][]byte{
		"CASE_TEST.txt":  []byte("same content"),      // Same content, different case
		"Mixed_Case.txt": []byte("also same content"), // Same content, different case
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// All files should be reported as duplicates (assuming case-insensitive comparison)
	expectedFilenames := []string{"case_test.txt", "CASE_TEST.txt", "mixed_CASE.txt", "Mixed_Case.txt"}
	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("failed to read CSV data")
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
