package e2e_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// // Test_DualFolder_NoDuplicates tests the case where there are no duplicate files between two folders
// func Test_DualFolder_NoDuplicates(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create two folders with different content
// 	folder1Files := map[string][]byte{
// 		"file1.txt":          []byte("content A"),
// 		"sub/file2.txt":      []byte("content B"),
// 		"sub/sub2/file3.txt": []byte("content C"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"fileX.txt":          []byte("content D"),
// 		"sub/fileY.txt":      []byte("content E"),
// 		"sub/sub2/fileZ.txt": []byte("content F"),
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// No duplicates expected, so no results file should be produced
// 	_, errFinal := readResultsFile(t, tempbinDir)

// 	if errFinal == nil {
// 		t.Errorf("Expected no results file to be produced, but file exists.")
// 	} else if !os.IsNotExist(errFinal) {
// 		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
// 	}
// }

// // Test_DualFolder_WithDuplicates tests the case where there are duplicate files between two folders
// func Test_DualFolder_WithDuplicates(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create source folder with various file types including duplicates
// 	sourceOptions := FileOptions{
// 		DuplicateFileCount: 2,
// 		DuplicatesPerFile:  0, // No duplicates within target folder
// 		UniqueFileCount:    3,
// 		FileTypes:          []FileType{TextFile, AudioFile},
// 		Prefix:             "source",
// 	}

// 	// Create target folder with some files that duplicate source files
// 	targetOptions := FileOptions{
// 		DuplicateFileCount: 2,
// 		DuplicatesPerFile:  0, // No duplicates within target folder
// 		UniqueFileCount:    3,
// 		FileTypes:          []FileType{TextFile, AudioFile},
// 		Prefix:             "target",
// 	}
// 	tempDir1, cleanup1 := createTestFiles(t, sourceOptions)
// 	tempDir2, cleanup2 := createTestFiles(t, targetOptions)

// 	// Now create identical files across folders
// 	// Create a file in source folder
// 	sharedContent1, err := createRandomContent(1024)
// 	if err != nil {
// 		t.Fatalf("failed to create random content: %v", err)
// 	}
// 	sharedFile1Source := filepath.Join(tempDir1, "shared-file1.txt")
// 	sharedFile1Target := filepath.Join(tempDir2, "identical-file1.txt")
// 	if err := createFileWithContent(sharedFile1Source, sharedContent1); err != nil {
// 		t.Fatalf("failed to create file %s: %v", sharedFile1Source, err)
// 	}
// 	if err := createFileWithContent(sharedFile1Target, sharedContent1); err != nil {
// 		t.Fatalf("failed to create file %s: %v", sharedFile1Target, err)
// 	}

// 	// Create another shared file with different content
// 	sharedContent2, err := createRandomContent(2048)
// 	if err != nil {
// 		t.Fatalf("failed to create random content: %v", err)
// 	}
// 	sharedFile2Source := filepath.Join(tempDir1, "shared-file2.pdf")
// 	sharedFile2Target := filepath.Join(tempDir2, "identical-file2.pdf")
// 	if err := createFileWithContent(sharedFile2Source, sharedContent2); err != nil {
// 		t.Fatalf("failed to create file %s: %v", sharedFile2Source, err)
// 	}
// 	if err := createFileWithContent(sharedFile2Target, sharedContent2); err != nil {
// 		t.Fatalf("failed to create file %s: %v", sharedFile2Target, err)
// 	}

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err = cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// We expect to find the shared files we created
// 	expectedFilenames := []string{
// 		"shared-file1.txt",
// 		"identical-file1.txt",
// 		"shared-file2.pdf",
// 		"identical-file2.pdf",
// 	}
// 	csvLines, err := readResultsFile(t, tempbinDir)

// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_EmptyFolders tests the case where both folders are empty
// func Test_DualFolder_EmptyFolders(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create two empty folders
// 	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{})
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{})

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// No duplicates expected for empty folders
// 	_, err = readResultsFile(t, tempbinDir)
// 	if err == nil {
// 		t.Error("Expected no results file for empty folders, but found one")
// 	}
// }

// // Test_DualFolder_HiddenFiles tests the case with hidden files that have duplicate content
// func Test_DualFolder_HiddenFiles(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create folders with hidden files
// 	folder1Files := map[string][]byte{
// 		"visible.txt":        []byte("unique content"),
// 		".hidden/secret.txt": []byte("duplicate content"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"normal.txt":          []byte("other content"),
// 		".invisible/data.txt": []byte("duplicate content"), // Same as secret.txt in folder1
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	expectedFilenames := []string{"secret.txt", "data.txt"}
// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_SpecialCharacters tests the case with files having special characters in their names
// func Test_DualFolder_SpecialCharacters(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create folders with files having special characters in names
// 	folder1Files := map[string][]byte{
// 		"file with spaces.txt": []byte("duplicate content"),
// 		"normal_file.txt":      []byte("unique content"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"file-with-special-!@#$%.txt": []byte("duplicate content"), // Same content as "file with spaces.txt"
// 		"regular.txt":                 []byte("different content"),
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	expectedFilenames := []string{"file with spaces.txt", "file-with-special-!@#$%.txt"}
// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_OneEmptyFolder tests the edge case where one folder is empty
// func Test_DualFolder_OneEmptyFolder_NoDuplicates(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	targetOptions := FileOptions{
// 		DuplicateFileCount: 2,
// 		DuplicatesPerFile:  0, // No duplicates within target folder
// 		UniqueFileCount:    3,
// 		FileTypes:          []FileType{TextFile, AudioFile},
// 		Prefix:             "target",
// 	}
// 	tempDir2, cleanup2 := createTestFiles(t, targetOptions)

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{}) // Empty folder

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// No duplicates expected when one folder is empty
// 	_, err = readResultsFile(t, tempbinDir)
// 	if err == nil {
// 		t.Error("Expected no results file when one folder is empty, but found one")
// 	}
// }

// // Test_DualFolder_OneEmptyFolder tests the edge case where one folder is empty
// func Test_DualFolder_OneEmptyFolder_WithDuplicates(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	targetOptions := FileOptions{
// 		DuplicateFileCount: 2,
// 		DuplicatesPerFile:  10, // No duplicates within target folder
// 		UniqueFileCount:    3,
// 		FileTypes:          []FileType{TextFile, AudioFile},
// 		Prefix:             "target",
// 	}
// 	tempDir2, cleanup2 := createTestFiles(t, targetOptions)

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{}) // Empty folder

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// No duplicates expected when one folder is empty
// 	allCSVLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsNumberOfRowsExpected(t, allCSVLines, targetOptions.CalculateTotalDuplicateFiles())
// }

// // Test_DualFolder_NestedStructure tests the edge case with complex nested directory structures
// func Test_DualFolder_NestedStructure(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create folders with complex nested structures
// 	folder1Files := map[string][]byte{
// 		"level1/file.txt":                    []byte("duplicate in deep structure"),
// 		"level1/level2/level3/deep_file.txt": []byte("another duplicate"),
// 		"level1/level2/unique.txt":           []byte("unique content"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"different/structure/file.txt":         []byte("duplicate in deep structure"), // Same as level1/file.txt
// 		"totally/different/path/deep_file.txt": []byte("another duplicate"),           // Same as level1/level2/level3/deep_file.txt
// 		"some/other/file.txt":                  []byte("different content"),
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	expectedFilenames := []string{
// 		"file.txt",
// 		"file.txt",
// 		"deep_file.txt",
// 		"deep_file.txt",
// 	}
// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_SameFilesButDifferentContent tests the edge case with files having the same names but different content
// func Test_DualFolder_SameFilesButDifferentContent(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create folders with same filenames but different content
// 	folder1Files := map[string][]byte{
// 		"same_name.txt":      []byte("content version A"),
// 		"also_same_name.txt": []byte("truly unique content"),
// 		"another_file.txt":   []byte("duplicate content"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"same_name.txt":      []byte("content version B"), // Same name, different content
// 		"also_same_name.txt": []byte("different content"), // Same name, different content
// 		"different_file.txt": []byte("duplicate content"), // Different name, same content as another_file.txt
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	// Only the files with same content should be reported as duplicates
// 	expectedFilenames := []string{"another_file.txt", "different_file.txt"}
// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("failed to read CSV data")
// 	}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_ParanoidMode tests that paranoid mode correctly identifies true duplicates between folders
// func Test_DualFolder_ParanoidMode(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create content for identical files
// 	content1, err := createRandomContent(4096) // 4KB identical content
// 	if err != nil {
// 		t.Fatalf("Failed to create random content: %v", err)
// 	}

// 	// Create content for MD5 collision simulation (different content, same hash)
// 	collisionFiles1 := []byte{
// 		0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
// 		0x2f, 0xca, 0xb5, 0x87, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89,
// 		// Additional bytes to make it larger
// 		0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc,
// 	}

// 	collisionFiles2 := []byte{
// 		0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
// 		0x2f, 0xca, 0xb5, 0x07, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89, // Note the 0x07 vs 0x87
// 		// Additional bytes to make it larger
// 		0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc,
// 	}

// 	folder1Files := map[string][]byte{
// 		"identical1.dat":       content1,
// 		"collision_source.dat": collisionFiles1,
// 		"unique1.dat":          []byte("This file is unique to folder 1"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"identical2.dat":       content1,        // Same as identical1.dat
// 		"collision_target.dat": collisionFiles2, // Different from collision_source.dat but hash collision
// 		"unique2.dat":          []byte("This file is unique to folder 2"),
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	// Run with paranoid mode
// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2, "-p")
// 	cmd.Stderr = &stderr

// 	err = cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("Failed to read CSV data")
// 	}

// 	// Only identical1.dat and identical2.dat should be found as duplicates
// 	expectedFilenames := []string{"identical1.dat", "identical2.dat"}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_UnicodeNormalization tests proper handling of Unicode filenames
// func Test_DualFolder_UnicodeNormalization(t *testing.T) {
// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create identical content
// 	content := []byte("Test content for unicode filename testing")

// 	// Create files with Unicode characters in different normalization forms
// 	folder1Files := map[string][]byte{
// 		"café.txt":        content, // é as single code point (NFC/composed)
// 		"normal-file.txt": content,
// 	}

// 	folder2Files := map[string][]byte{
// 		"cafe\u0301.txt":  content, // é as 'e' + combining acute accent (NFD/decomposed)
// 		"другой-файл.txt": content, // Cyrillic characters
// 	}

// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	defer func() {
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}

// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("Failed to read CSV data")
// 	}

// 	// All files should be found as duplicates because they have same content
// 	expectedFilenames := []string{"café.txt", "cafe\u0301.txt", "normal-file.txt", "другой-файл.txt"}
// 	csvContainsExpected(t, csvLines, expectedFilenames)
// }

// // Test_DualFolder_PermissionDenied tests handling of permission errors
// func Test_DualFolder_PermissionDenied_ShouldNotFail(t *testing.T) {
// 	// Skip on Windows as permissions work differently
// 	if runtime.GOOS == "windows" {
// 		t.Skip("Skipping permissions test on Windows")
// 	}

// 	var stderr bytes.Buffer

// 	binaryPath, tempbinDir, cleanupBin := buildBinary(t)

// 	// Create folders with normal files
// 	folder1Files := map[string][]byte{
// 		"readable1.txt": []byte("content A"),
// 		"readable2.txt": []byte("content B"),
// 	}

// 	folder2Files := map[string][]byte{
// 		"readable3.txt": []byte("content A"), // Identical to readable1.txt
// 		"readable4.txt": []byte("content C"),
// 	}

// 	// Create test directories with files
// 	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
// 	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

// 	// Create an unreadable file in folder2
// 	unreadableFile := filepath.Join(tempDir2, "unreadable.txt")
// 	err := os.WriteFile(unreadableFile, []byte("secret content"), 0000)
// 	if err != nil {
// 		t.Fatalf("Failed to create unreadable file: %v", err)
// 	}

// 	defer func() {
// 		// Make the file readable again so it can be deleted
// 		os.Chmod(unreadableFile, 0644)
// 		cleanup1()
// 		cleanup2()
// 		cleanupBin()
// 	}()

// 	cmd := exec.Command(binaryPath, "-s="+tempDir1, "-t="+tempDir2)
// 	cmd.Stderr = &stderr

// 	err = cmd.Run()
// 	// The app should still run successfully despite permission errors
// 	if err != nil {
// 		t.Fatalf("CLI app failed with error: %v, Stderr: %s", err, stderr.String())
// 	}
// 	csvLines, err := readResultsFile(t, tempbinDir)
// 	if err != nil {
// 		t.Fatal("Failed to read CSV data")
// 	}

//		// The readable duplicates should still be found
//		expectedFilenames := []string{"readable1.txt", "readable3.txt"}
//		csvContainsExpected(t, csvLines, expectedFilenames)
//	}
func Test_DualFolder_NoDuplicates(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create two folders with different content
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
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)
	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2, // New: Specify TargetDir
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true, // Crucial: Enable dual mode
	}

	// 4. Execute the logic directly
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: No duplicates expected
	_, errFinal := readResultsFile(t, testResultsDir)

	if errFinal == nil {
		t.Errorf("Expected no results file to be produced, but file exists.")
	} else if !os.IsNotExist(errFinal) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
	}
}

func Test_DualFolder_WithDuplicates(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// ... (FileOptions definitions are unchanged) ...
	sourceOptions := FileOptions{DuplicateFileCount: 2, DuplicatesPerFile: 0, UniqueFileCount: 3, FileTypes: []FileType{TextFile, AudioFile}, Prefix: "source"}
	targetOptions := FileOptions{DuplicateFileCount: 2, DuplicatesPerFile: 0, UniqueFileCount: 3, FileTypes: []FileType{TextFile, AudioFile}, Prefix: "target"}

	tempDir1, cleanup1 := createTestFiles(t, sourceOptions)
	tempDir2, cleanup2 := createTestFiles(t, targetOptions)

	// Create identical files across folders (unchanged logic)
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

	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err = app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{
		"shared-file1.txt", "identical-file1.txt",
		"shared-file2.pdf", "identical-file2.pdf",
	}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_EmptyFolders(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create two empty folders
	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{})
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{})
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")

	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)
	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)

	// A. Check if an error was returned at all
	if err == nil {
		t.Fatalf("Expected execution to return the 'no files found' error, but it succeeded without error.")
	}

	// B. Check if the returned error matches the expected custom error
	if !errors.Is(err, processing.ErrNoFilesFound) {
		t.Fatalf("Expected error type %v, but got: %v", processing.ErrNoFilesFound, err)
	}

	// C. (Optional but good practice) Verify no results file was written
	_, errFile := readResultsFile(t, testResultsDir)
	if errFile == nil {
		t.Error("Expected no results file (since no files were processed), but found one.")
	} else if !os.IsNotExist(errFile) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFile)
	}
}
func Test_DualFolder_HiddenFiles(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create folders with hidden files
	folder1Files := map[string][]byte{
		"visible.txt":        []byte("unique content"),
		".hidden/secret.txt": []byte("duplicate content"),
	}
	folder2Files := map[string][]byte{
		"normal.txt":          []byte("other content"),
		".invisible/data.txt": []byte("duplicate content"), // Same as secret.txt
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{"secret.txt", "data.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_SpecialCharacters(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create folders with files having special characters
	folder1Files := map[string][]byte{
		"file with spaces.txt": []byte("duplicate content"),
		"normal_file.txt":      []byte("unique content"),
	}
	folder2Files := map[string][]byte{
		"file-with-special-!@#$%.txt": []byte("duplicate content"),
		"regular.txt":                 []byte("different content"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{"file with spaces.txt", "file-with-special-!@#$%.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_OneEmptyFolder_NoDuplicates(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	targetOptions := FileOptions{
		DuplicateFileCount: 0, // Ensure no duplicates
		DuplicatesPerFile:  0,
		UniqueFileCount:    5,
		FileTypes:          []FileType{TextFile},
	}
	tempDir2, cleanup2 := createTestFiles(t, targetOptions)
	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{}) // Empty folder
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1, // Empty
		TargetDir:             tempDir2, // Has unique files
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: No duplicates expected
	_, err = readResultsFile(t, testResultsDir)
	if err == nil {
		t.Error("Expected no results file when one folder is empty, but found one")
	}
}
func Test_DualFolder_OneEmptyFolder_WithDuplicates(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// targetOptions is configured to create duplicates *within* tempDir2
	targetOptions := FileOptions{
		DuplicateFileCount: 2,
		DuplicatesPerFile:  10,
		UniqueFileCount:    3,
		FileTypes:          []FileType{TextFile, AudioFile},
		Prefix:             "target",
	}
	tempDir2, cleanup2 := createTestFiles(t, targetOptions)
	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{}) // Empty folder
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1, // Empty
		TargetDir:             tempDir2, // Has internal duplicates
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: Expecting duplicates from *within* tempDir2
	allCSVLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}
	// Verify that the internal duplicates in tempDir2 were found
	csvContainsNumberOfRowsExpected(t, allCSVLines, targetOptions.CalculateTotalDuplicateFiles())
}
func Test_DualFolder_NestedStructure(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create folders with complex nested structures
	folder1Files := map[string][]byte{
		"level1/file.txt":                    []byte("duplicate in deep structure"),
		"level1/level2/level3/deep_file.txt": []byte("another duplicate"),
		"level1/level2/unique.txt":           []byte("unique content"),
	}
	folder2Files := map[string][]byte{
		"different/structure/file.txt":         []byte("duplicate in deep structure"),
		"totally/different/path/deep_file.txt": []byte("another duplicate"),
		"some/other/file.txt":                  []byte("different content"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: Expecting the four duplicate files
	expectedFilenames := []string{"file.txt", "file.txt", "deep_file.txt", "deep_file.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_SameFilesButDifferentContent(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create folders with same filenames but different content
	folder1Files := map[string][]byte{
		"same_name.txt":      []byte("content version A"),
		"also_same_name.txt": []byte("truly unique content"),
		"another_file.txt":   []byte("duplicate content"),
	}
	folder2Files := map[string][]byte{
		"same_name.txt":      []byte("content version B"), // Same name, different content (ignored)
		"also_same_name.txt": []byte("different content"), // Same name, different content (ignored)
		"different_file.txt": []byte("duplicate content"), // Different name, same content (found)
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: Only the two files with identical content are reported
	expectedFilenames := []string{"another_file.txt", "different_file.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_ParanoidMode(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// ... (File content setup is unchanged) ...
	content1, _ := createRandomContent(4096)
	collisionFiles1 := []byte{ /* ... bytes ... */ }
	collisionFiles2 := []byte{ /* ... bytes ... */ }

	folder1Files := map[string][]byte{
		"identical1.dat":       content1,
		"collision_source.dat": collisionFiles1, // Hash collision
		"unique1.dat":          []byte("This file is unique to folder 1"),
	}
	folder2Files := map[string][]byte{
		"identical2.dat":       content1,        // True duplicate
		"collision_target.dat": collisionFiles2, // Hash collision, but content differs
		"unique2.dat":          []byte("This file is unique to folder 2"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments: Crucially enable ParanoidMode
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
		ParanoidMode:          true, // <-- Enable full byte-by-byte comparison
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: Only identical1.dat and identical2.dat are true duplicates
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}

	expectedFilenames := []string{"identical1.dat", "identical2.dat"}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_UnicodeNormalization(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create files with Unicode characters in different normalization forms
	content := []byte("Test content for unicode filename testing")
	folder1Files := map[string][]byte{
		"café.txt":        content, // NFC (composed)
		"normal-file.txt": content,
	}
	folder2Files := map[string][]byte{
		"cafe\u0301.txt":  content, // NFD (decomposed) - same content as café.txt
		"другой-файл.txt": content, // Cyrillic characters - same content as normal-file.txt
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)
	defer func() { cleanup1(); cleanup2() }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{"café.txt", "cafe\u0301.txt", "normal-file.txt", "другой-файл.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
func Test_DualFolder_PermissionDenied_ShouldNotFail(t *testing.T) {
	// Skip on Windows as permissions work differently (unchanged)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permissions test on Windows")
	}

	// 1. New App instance
	app := setupTestApp(t)

	// 2. Create test directories and files
	folder1Files := map[string][]byte{"readable1.txt": []byte("content A"), "readable2.txt": []byte("content B")}
	folder2Files := map[string][]byte{"readable3.txt": []byte("content A"), "readable4.txt": []byte("content C")}

	tempDir1, cleanup1 := createTestFilesByteArray(t, folder1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, folder2Files)

	// Create an unreadable file in folder2 (unchanged)
	unreadableFile := filepath.Join(tempDir2, "unreadable.txt")
	err := os.WriteFile(unreadableFile, []byte("secret content"), 0000)
	if err != nil {
		t.Fatalf("Failed to create unreadable file: %v", err)
	}

	defer func() {
		// Make the file readable again so it can be deleted (unchanged)
		os.Chmod(unreadableFile, 0644)
		cleanup1()
		cleanup2()
	}()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		SourceDir:             tempDir1,
		TargetDir:             tempDir2,
		ResultsDir:            testResultsDir,
		CacheDir:              testCacheDir,
		CPUs:                  1,
		BufSize:               1024,
		DualFolderModeEnabled: true,
	}

	// 4. Execute the logic
	err = app.StartExecution(args)
	// The app should still run successfully despite permission errors encountered during file walk/hash
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: The readable duplicates should still be found
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data")
	}
	expectedFilenames := []string{"readable1.txt", "readable3.txt"}
	csvContainsExpected(t, csvLines, expectedFilenames)
}
