package e2e_tests

import (
	"DuDe/internal/models"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func Test_SingleFolder_EmptyFolder(t *testing.T) {
	// 1. new App instance
	app := setupTestApp(t)
	// 2. Create files (empty here)
	tempDir, cleanup := createTestFilesByteArray(t, map[string][]byte{})

	defer func() { cleanup(); deleteTestFolder(t) }()
	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")

	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}

	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// Check the results file path defined in the arguments (not the binary's directory)
	_, err = readResultsFile(t, testResultsDir)

	if err == nil {
		t.Error("Expected no results file for empty folder, but found one")
	}
}

func Test_SingleFolder_NoDuplicates(t *testing.T) {
	// 1. new App instance
	app := setupTestApp(t)
	// 2. Create files (empty here)
	files := map[string][]byte{
		"file1.txt":          []byte("content A"),
		"sub/file2.txt":      []byte("content B"),
		"sub/sub2/file3.txt": []byte("content C"),
	}
	tempDir, cleanup := createTestFilesByteArray(t, files)

	defer func() { cleanup(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")

	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	_, errFinal := readResultsFile(t, testResultsDir)

	if errFinal == nil {
		t.Errorf("Expected no results file to be produced, but file exists.")
	} else if !errors.Is(errFinal, os.ErrNotExist) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
	}
}

func Test_SingleFolder_WithDuplicates(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	options := FileOptions{
		DuplicateFileCount: 2,
		DuplicatesPerFile:  1,
		UniqueFileCount:    2,
		FileTypes:          []FileType{TextFile, AudioFile},
		Prefix:             "source",
	}

	// 2. Create complex test files using the helper function
	tempDir, cleanup := createTestFiles(t, options)
	defer func() { cleanup(); deleteTestFolder(t) }()

	// 3. Define the output directories within the temporary test scope
	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 4. Prepare Arguments
	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}

	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 6. Verification
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}

	// Assert the correct number of duplicate files were found
	csvContainsNumberOfRowsExpected(t, csvLines, options.CalculateTotalDuplicateFiles())
}

func Test_SingleFolder_HiddenFiles(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	files := map[string][]byte{
		"file1.txt":         []byte("duplicate content"),
		".hidden/file2.txt": []byte("duplicate content"),
		"file3.txt":         []byte("unique content"),
	}

	// 2. Create test files
	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() { cleanup(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{"file1.txt", "file2.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	// Assert that both duplicate files (one hidden) are in the results
	csvContainsExpected(t, csvLines, expectedFilenames)
}

func Test_SingleFolder_SpecialCharacters(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	files := map[string][]byte{
		"file with spaces.txt":        []byte("duplicate content"),
		"file-with-special-!@#$%.txt": []byte("duplicate content"),
		"normal_file.txt":             []byte("unique content"),
	}

	// 2. Create test files
	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() { cleanup(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
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

func Test_SingleFolder_DifferentSizes(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// Create files with same content but different sizes
	files := map[string][]byte{
		"small.txt":  []byte("content"),                              // Unique size/content
		"large1.txt": []byte("content" + string(make([]byte, 1024))), // Same content/size as large2
		"large2.txt": []byte("content" + string(make([]byte, 1024))), // Same content/size as large1
	}

	// 2. Create test files
	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() { cleanup(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments
	args := models.ExecutionParams{
		Mode:            models.ExecutionModeDuplicates,
		Directories:     []string{tempDir},
		ResultsDir:      testResultsDir,
		CacheDir:        testCacheDir,
		WorkerCount:     1,
		BufferSizeBytes: 1024,
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification
	expectedFilenames := []string{"large1.txt", "large2.txt"}
	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	// Only the two large identical files should be listed
	csvContainsExpected(t, csvLines, expectedFilenames)
}
