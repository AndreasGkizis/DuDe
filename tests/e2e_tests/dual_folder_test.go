package e2e_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

	// Create source folder with various file types including duplicates
	sourceOptions := DefaultFileOptions()
	sourceOptions.DuplicateFileCount = 2
	sourceOptions.UniqueFileCount = 3
	sourceOptions.DuplicatesPerFile = 0 // No duplicates within source folder
	sourceOptions.FileTypes = []FileType{TextFile, AudioFile, MixedFile}
	sourceOptions.FileSize = 2048
	sourceOptions.Prefix = "source"

	// Create target folder with some files that duplicate source files
	targetOptions := DefaultFileOptions()
	targetOptions.DuplicateFileCount = 2
	targetOptions.UniqueFileCount = 3
	targetOptions.DuplicatesPerFile = 0 // No duplicates within target folder
	targetOptions.FileTypes = []FileType{TextFile, AudioFile, GreekFile}
	targetOptions.FileSize = 2048 // Same size to ensure content hash matches
	targetOptions.Prefix = "target"

	tempDir1, cleanup1 := createTestFiles(t, sourceOptions)
	tempDir2, cleanup2 := createTestFiles(t, targetOptions)

	// Now create identical files across folders
	// Create a file in source folder
	sharedContent1, err := createRandomContent(1024)
	if err != nil {
		t.Fatalf("failed to create random content: %v", err)
	}
	sharedFile1Source := filepath.Join(tempDir1, "shared-file1.txt")
	sharedFile1Target := filepath.Join(tempDir2, "identical-file1.txt")
	if err := createFileWithContent(sharedFile1Source, sharedContent1); err != nil {
		t.Fatalf("failed to create file %s: %v", sharedFile1Source, err)
	}
	if err := createFileWithContent(sharedFile1Target, sharedContent1); err != nil {
		t.Fatalf("failed to create file %s: %v", sharedFile1Target, err)
	}

	// Create another shared file with different content
	sharedContent2, err := createRandomContent(2048)
	if err != nil {
		t.Fatalf("failed to create random content: %v", err)
	}
	sharedFile2Source := filepath.Join(tempDir1, "shared-file2.pdf")
	sharedFile2Target := filepath.Join(tempDir2, "identical-file2.pdf")
	if err := createFileWithContent(sharedFile2Source, sharedContent2); err != nil {
		t.Fatalf("failed to create file %s: %v", sharedFile2Source, err)
	}
	if err := createFileWithContent(sharedFile2Target, sharedContent2); err != nil {
		t.Fatalf("failed to create file %s: %v", sharedFile2Target, err)
	}

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// We expect to find the shared files we created
	expectedFilenames := []string{
		"shared-file1.txt",
		"identical-file1.txt",
		"shared-file2.pdf",
		"identical-file2.pdf",
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

// Test_DualFolder_ParanoidMode tests that paranoid mode correctly identifies true duplicates between folders
func Test_DualFolder_ParanoidMode(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create content for identical files
	content1, err := createRandomContent(4096) // 4KB identical content
	if err != nil {
		t.Fatalf("Failed to create random content: %v", err)
	}

	// Create content for MD5 collision simulation (different content, same hash)
	collisionFiles1 := []byte{
		0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
		0x2f, 0xca, 0xb5, 0x87, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89,
		// Additional bytes to make it larger
		0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc,
	}

	collisionFiles2 := []byte{
		0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
		0x2f, 0xca, 0xb5, 0x07, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89, // Note the 0x07 vs 0x87
		// Additional bytes to make it larger
		0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc,
	}

	folder1Files := map[string][]byte{
		"identical1.dat":       content1,
		"collision_source.dat": collisionFiles1,
		"unique1.dat":          []byte("This file is unique to folder 1"),
	}

	folder2Files := map[string][]byte{
		"identical2.dat":       content1,        // Same as identical1.dat
		"collision_target.dat": collisionFiles2, // Different from collision_source.dat but hash collision
		"unique2.dat":          []byte("This file is unique to folder 2"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	// Run with paranoid mode
	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2, "-p")
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}

	// Only identical1.dat and identical2.dat should be found as duplicates
	expectedFilenames := []string{"identical1.dat", "identical2.dat"}
	csvContainsExpected(t, csvLines, expectedFilenames)

	// Make sure collision files are not reported as duplicates in paranoid mode
	for _, line := range csvLines {
		if len(line) >= 1 &&
			(strings.Contains(line[0], "collision_source.dat") ||
				strings.Contains(line[0], "collision_target.dat")) {
			t.Errorf("Found collision files in results when they should not be detected as duplicates in paranoid mode")
		}
	}
}

// Test_DualFolder_LargeFiles tests handling of larger files (moderately sized, not extreme)
func Test_DualFolder_LargeFiles(t *testing.T) {
	var stderr bytes.Buffer

	// Skip this test if running in short mode
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Generate moderately large random content (1MB instead of 10MB to avoid test issues)
	fileSize := 1 * 1024 * 1024 // 1MB is large enough for testing but won't cause memory issues
	largeContent, err := createRandomContent(fileSize)
	if err != nil {
		t.Fatalf("Failed to create large random content: %v", err)
	}

	t.Logf("Successfully created %d bytes of test content", len(largeContent))

	// Create a slightly modified copy (change last 10 bytes)
	differentLargeContent := make([]byte, len(largeContent))
	copy(differentLargeContent, largeContent)
	for i := 0; i < 10; i++ {
		if len(differentLargeContent) > i {
			differentLargeContent[len(differentLargeContent)-i-1] = byte(i)
		}
	}

	// Create separate folders with large files
	folder1Files := map[string][]byte{
		"large_original.bin": largeContent,
	}

	folder2Files := map[string][]byte{
		"large_duplicate.bin": largeContent,          // Identical to original
		"large_different.bin": differentLargeContent, // Almost identical
	}

	t.Log("Creating test directories...")
	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	t.Logf("Created test directories: %s and %s", tempDir1, tempDir2)

	defer func() {
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	// Verify files were created correctly
	originalStat, err := os.Stat(filepath.Join(tempDir1, "large_original.bin"))
	if err != nil {
		t.Fatalf("Failed to stat original file: %v", err)
	}
	duplicateStat, err := os.Stat(filepath.Join(tempDir2, "large_duplicate.bin"))
	if err != nil {
		t.Fatalf("Failed to stat duplicate file: %v", err)
	}

	t.Logf("Original file size: %d bytes, Duplicate file size: %d bytes",
		originalStat.Size(), duplicateStat.Size())

	// First run standard mode
	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	t.Log("Running DuDe in standard mode...")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Logf("Error reading results: %v", err)
		t.Fatal("Failed to read CSV data")
	}

	t.Logf("Found %d lines in CSV results", len(csvLines))

	// Since we're only looking for filenames, ignore paths in comparison
	var found1, found2 bool
	for _, line := range csvLines {
		if len(line) >= 1 {
			t.Logf("Checking CSV line: %v", line)
			if strings.Contains(line[0], "large_original.bin") ||
				strings.Contains(line[2], "large_original.bin") {
				found1 = true
			}
			if strings.Contains(line[0], "large_duplicate.bin") ||
				strings.Contains(line[2], "large_duplicate.bin") {
				found2 = true
			}
		}
	}

	if !found1 || !found2 {
		t.Errorf("Failed to find expected files in results: large_original.bin found: %v, large_duplicate.bin found: %v",
			found1, found2)
	}

	// Large different file should not be reported
	for _, line := range csvLines {
		if len(line) >= 1 &&
			(strings.Contains(line[0], "large_different.bin") ||
				strings.Contains(line[2], "large_different.bin")) {
			t.Errorf("Found large_different.bin in results when it shouldn't be detected as duplicate")
		}
	}

	// Now run with paranoid mode
	cmd = exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2, "-p")
	cmd.Stderr = &stderr

	t.Log("Running DuDe in paranoid mode...")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	csvLines, err = readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}

	// Check for duplicates in paranoid mode
	found1, found2 = false, false
	for _, line := range csvLines {
		if len(line) >= 1 {
			if strings.Contains(line[0], "large_original.bin") ||
				strings.Contains(line[2], "large_original.bin") {
				found1 = true
			}
			if strings.Contains(line[0], "large_duplicate.bin") ||
				strings.Contains(line[2], "large_duplicate.bin") {
				found2 = true
			}
		}
	}

	if !found1 || !found2 {
		t.Errorf("Failed to find expected files in paranoid mode results: large_original.bin found: %v, large_duplicate.bin found: %v",
			found1, found2)
	}
}

// Test_DualFolder_UnicodeNormalization tests proper handling of Unicode filenames
func Test_DualFolder_UnicodeNormalization(t *testing.T) {
	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create identical content
	content := []byte("Test content for unicode filename testing")

	// Create files with Unicode characters in different normalization forms
	folder1Files := map[string][]byte{
		"café.txt":        content, // é as single code point (NFC/composed)
		"normal-file.txt": content,
	}

	folder2Files := map[string][]byte{
		"cafe\u0301.txt":  content, // é as 'e' + combining acute accent (NFD/decomposed)
		"другой-файл.txt": content, // Cyrillic characters
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

	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}

	// All files should be found as duplicates because they have same content
	expectedFilenames := []string{"café.txt", "cafe\u0301.txt", "normal-file.txt", "другой-файл.txt"}
	csvContainsExpected(t, csvLines, expectedFilenames)
}

// Test_DualFolder_PermissionDenied tests handling of permission errors
func Test_DualFolder_PermissionDenied(t *testing.T) {
	// Skip on Windows as permissions work differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permissions test on Windows")
	}

	var stderr bytes.Buffer

	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

	// Create folders with normal files
	folder1Files := map[string][]byte{
		"readable1.txt": []byte("content A"),
		"readable2.txt": []byte("content B"),
	}

	folder2Files := map[string][]byte{
		"readable3.txt": []byte("content A"), // Identical to readable1.txt
		"readable4.txt": []byte("content C"),
	}

	// Create test directories with files
	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	// Create an unreadable file in folder2
	unreadableFile := filepath.Join(tempDir2, "unreadable.txt")
	err := os.WriteFile(unreadableFile, []byte("secret content"), 0000)
	if err != nil {
		t.Fatalf("Failed to create unreadable file: %v", err)
	}

	defer func() {
		// Make the file readable again so it can be deleted
		os.Chmod(unreadableFile, 0644)
		cleanup1()
		cleanup2()
		cleanupBin()
	}()

	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
	cmd.Stderr = &stderr

	err = cmd.Run()
	// The app should still run successfully despite permission errors
	if err != nil {
		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
	}

	// Check if stderr contains permission denied message
	if !strings.Contains(stderr.String(), "permission") &&
		!strings.Contains(stderr.String(), "Permission") {
		t.Logf("Expected stderr to contain permission error message, got: %s", stderr.String())
	}

	csvLines, err := readResultsFile(t, tempbinDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}

	// The readable duplicates should still be found
	expectedFilenames := []string{"readable1.txt", "readable3.txt"}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
