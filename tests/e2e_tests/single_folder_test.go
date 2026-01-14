package e2e_tests

import (
	"DuDe/internal/models"
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
	}
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	_, errFinal := readResultsFile(t, testResultsDir)

	if errFinal == nil {
		t.Errorf("Expected no results file to be produced, but file exists.")
	} else if !os.IsNotExist(errFinal) {
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
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
		SourceDir:  tempDir,
		ResultsDir: testResultsDir,
		CacheDir:   testCacheDir,
		CPUs:       1,
		BufSize:    1024,
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

func Test_SingleFolder_ShouldNotInclude_If_Md5Collision(t *testing.T) {
	// 1. New App instance
	app := setupTestApp(t)

	// Files with different content designed to produce an MD5 collision
	files := map[string][]byte{
		"file1.txt": {
			0xd1, 0x31, 0xdd, 0x02, 0xc5, 0xe6, 0xee, 0xc4, 0x69, 0x3d, 0x9a, 0x06, 0x98, 0xaf, 0xf9, 0x5c,
			0x2f, 0xca, 0xb5, 0x87, 0x12, 0x46, 0x7e, 0xab, 0x40, 0x04, 0x58, 0x3e, 0xb8, 0xfb, 0x7f, 0x89,
			0x55, 0xad, 0x34, 0x06, 0x09, 0xf4, 0xb3, 0x02, 0x83, 0xe4, 0x88, 0x83, 0x25, 0x71, 0x41, 0x5a,
			0x08, 0x51, 0x25, 0xe8, 0xf7, 0xcd, 0xc9, 0x9f, 0xd9, 0x1d, 0xbd, 0xf2, 0x80, 0x37, 0x3c, 0x5b,
			0xd8, 0x82, 0x3e, 0x31, 0x56, 0x34, 0x8f, 0x5b, 0xae, 0x6d, 0xac, 0xd4, 0x36, 0xc9, 0x19, 0xc6,
			0xdd, 0x53, 0xe2, 0xb4, 0x87, 0xda, 0x03, 0xfd, 0x02, 0x39, 0x63, 0x06, 0xd2, 0x48, 0xcd, 0xa0,
			0xe9, 0x9f, 0x33, 0x42, 0x0f, 0x57, 0x7e, 0xe8, 0xce, 0x54, 0xb6, 0x70, 0x80, 0xa8, 0x0d, 0x1e,
			0xc6, 0x98, 0x21, 0xbc, 0xb6, 0xa8, 0x83, 0x93, 0x96, 0xf9, 0x65, 0x2b, 0x6f, 0xf7, 0x2a, 0x70,
		},
		"file2.txt": {
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

	// 2. Create test files
	tempDir, cleanup := createTestFilesByteArray(t, files)
	defer func() { cleanup(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(tempDir, "results")
	testCacheDir := filepath.Join(tempDir, "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	// 3. Prepare Arguments: Crucially enable ParanoidMode
	args := models.ExecutionParams{
		SourceDir:    tempDir,
		ResultsDir:   testResultsDir,
		CacheDir:     testCacheDir,
		CPUs:         1,
		BufSize:      1024,
		ParanoidMode: true, // <-- Enable full byte-by-byte comparison
	}

	// 4. Execute the logic
	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("E2E app failed with error: %v", err)
	}

	// 5. Verification: Because content differs (despite hash collision), no duplicates should be found.
	_, errFinal := readResultsFile(t, testResultsDir)

	if errFinal == nil {
		t.Errorf("Expected no results file (due to content mismatch in paranoid mode), but file exists.")
	} else if !os.IsNotExist(errFinal) {
		t.Errorf("Expected results file to not exist, but got unexpected error: %v", errFinal)
	}
}
