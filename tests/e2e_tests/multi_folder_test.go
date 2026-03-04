package e2e_tests

import (
	"DuDe/internal/models"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Test_MultiFolder_NoDuplicates verifies that three directories with entirely unique
// content produce no duplicates.
func Test_MultiFolder_NoDuplicates(t *testing.T) {
	app := setupTestApp(t)

	dir1Files := map[string][]byte{
		"a.txt": []byte("content A"),
		"b.txt": []byte("content B"),
	}
	dir2Files := map[string][]byte{
		"c.txt": []byte("content C"),
		"d.txt": []byte("content D"),
	}
	dir3Files := map[string][]byte{
		"e.txt": []byte("content E"),
		"f.txt": []byte("content F"),
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, dir1Files)
	tempDir2, cleanup2 := createTestFilesByteArray(t, dir2Files)
	tempDir3, cleanup3 := createTestFilesByteArray(t, dir3Files)
	defer func() { cleanup1(); cleanup2(); cleanup3(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Directories: []string{tempDir1, tempDir2, tempDir3},
		ResultsDir:  testResultsDir,
		CacheDir:    testCacheDir,
		CPUs:        1,
		BufSize:     1024,
	}

	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("app failed: %v", err)
	}

	_, errFinal := readResultsFile(t, testResultsDir)
	if errFinal == nil {
		t.Error("Expected no results file for 3 dirs with unique content, but found one")
	} else if !errors.Is(errFinal, os.ErrNotExist) {
		t.Errorf("Expected results file to not exist, got unexpected error: %v", errFinal)
	}
}

// Test_MultiFolder_DuplicatesAcrossAllThree verifies that the same file present in all
// three directories is detected as a duplicate.
func Test_MultiFolder_DuplicatesAcrossAllThree(t *testing.T) {
	app := setupTestApp(t)

	sharedContent, err := createRandomContent(2048)
	if err != nil {
		t.Fatalf("failed to create random content: %v", err)
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{"unique1.txt": []byte("unique A")})
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{"unique2.txt": []byte("unique B")})
	tempDir3, cleanup3 := createTestFilesByteArray(t, map[string][]byte{"unique3.txt": []byte("unique C")})
	defer func() { cleanup1(); cleanup2(); cleanup3(); deleteTestFolder(t) }()

	// Plant the same file in all three directories
	if err := createFileWithContent(filepath.Join(tempDir1, "shared.dat"), sharedContent); err != nil {
		t.Fatalf("create shared file in dir1: %v", err)
	}
	if err := createFileWithContent(filepath.Join(tempDir2, "shared_copy.dat"), sharedContent); err != nil {
		t.Fatalf("create shared file in dir2: %v", err)
	}
	if err := createFileWithContent(filepath.Join(tempDir3, "shared_another.dat"), sharedContent); err != nil {
		t.Fatalf("create shared file in dir3: %v", err)
	}

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Directories: []string{tempDir1, tempDir2, tempDir3},
		ResultsDir:  testResultsDir,
		CacheDir:    testCacheDir,
		CPUs:        1,
		BufSize:     1024,
	}

	err = app.StartExecution(args)
	if err != nil {
		t.Fatalf("app failed: %v", err)
	}

	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsExpected(t, csvLines, []string{"shared.dat", "shared_copy.dat", "shared_another.dat"})
}

// Test_MultiFolder_DuplicatesInSubset verifies that a duplicate between only two of
// three directories is correctly detected while the third directory is unaffected.
func Test_MultiFolder_DuplicatesInSubset(t *testing.T) {
	app := setupTestApp(t)

	sharedContent, err := createRandomContent(1024)
	if err != nil {
		t.Fatalf("failed to create random content: %v", err)
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{"unique1.txt": []byte("only in dir1")})
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{"unique2.txt": []byte("only in dir2")})
	// dir3 has completely different content
	tempDir3, cleanup3 := createTestFilesByteArray(t, map[string][]byte{"unique3.txt": []byte("only in dir3")})
	defer func() { cleanup1(); cleanup2(); cleanup3(); deleteTestFolder(t) }()

	// Plant the same file only in dir1 and dir3 (NOT dir2)
	if err := createFileWithContent(filepath.Join(tempDir1, "dup_in_1_and_3.bin"), sharedContent); err != nil {
		t.Fatalf("create in dir1: %v", err)
	}
	if err := createFileWithContent(filepath.Join(tempDir3, "dup_in_1_and_3_copy.bin"), sharedContent); err != nil {
		t.Fatalf("create in dir3: %v", err)
	}

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Directories: []string{tempDir1, tempDir2, tempDir3},
		ResultsDir:  testResultsDir,
		CacheDir:    testCacheDir,
		CPUs:        1,
		BufSize:     1024,
	}

	err = app.StartExecution(args)
	if err != nil {
		t.Fatalf("app failed: %v", err)
	}

	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	// Only the two files from dir1 and dir3 should appear
	csvContainsExpected(t, csvLines, []string{"dup_in_1_and_3.bin", "dup_in_1_and_3_copy.bin"})
	// Unique files from dir2 must NOT appear in results
	for _, line := range csvLines {
		for _, field := range line {
			if field == "unique2.txt" {
				t.Error("unique2.txt from dir2 should not appear in duplicate results")
			}
		}
	}
}

// Test_MultiFolder_SomeDirectoriesEmpty verifies that two empty directories alongside
// one directory with internal duplicates still correctly reports the duplicates.
func Test_MultiFolder_SomeDirectoriesEmpty(t *testing.T) {
	app := setupTestApp(t)

	options := FileOptions{
		DuplicateFileCount: 2,
		DuplicatesPerFile:  1,
		UniqueFileCount:    2,
		FileTypes:          []FileType{TextFile},
		Prefix:             "filled",
	}
	filledDir, cleanupFilled := createTestFiles(t, options)
	emptyDir1, cleanupEmpty1 := createTestFilesByteArray(t, map[string][]byte{})
	emptyDir2, cleanupEmpty2 := createTestFilesByteArray(t, map[string][]byte{})
	defer func() { cleanupFilled(); cleanupEmpty1(); cleanupEmpty2(); deleteTestFolder(t) }()

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Directories: []string{filledDir, emptyDir1, emptyDir2},
		ResultsDir:  testResultsDir,
		CacheDir:    testCacheDir,
		CPUs:        1,
		BufSize:     1024,
	}

	err := app.StartExecution(args)
	if err != nil {
		t.Fatalf("app failed: %v", err)
	}

	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	csvContainsNumberOfRowsExpected(t, csvLines, options.CalculateTotalDuplicateFiles())
}

// Test_MultiFolder_FourDirectories verifies that duplicate detection works correctly
// across four directories with overlapping content.
func Test_MultiFolder_FourDirectories(t *testing.T) {
	app := setupTestApp(t)

	sharedContent, err := createRandomContent(4096)
	if err != nil {
		t.Fatalf("failed to create random content: %v", err)
	}

	tempDir1, cleanup1 := createTestFilesByteArray(t, map[string][]byte{"unique1.txt": []byte("A")})
	tempDir2, cleanup2 := createTestFilesByteArray(t, map[string][]byte{"unique2.txt": []byte("B")})
	tempDir3, cleanup3 := createTestFilesByteArray(t, map[string][]byte{"unique3.txt": []byte("C")})
	tempDir4, cleanup4 := createTestFilesByteArray(t, map[string][]byte{"unique4.txt": []byte("D")})
	defer func() { cleanup1(); cleanup2(); cleanup3(); cleanup4(); deleteTestFolder(t) }()

	// same file in dirs 1, 2 and 4
	for _, pair := range []struct {
		dir, name string
	}{
		{tempDir1, "common.bin"},
		{tempDir2, "common_again.bin"},
		{tempDir4, "common_yet_again.bin"},
	} {
		if err := createFileWithContent(filepath.Join(pair.dir, pair.name), sharedContent); err != nil {
			t.Fatalf("create file %s: %v", pair.name, err)
		}
	}

	testResultsDir := filepath.Join(t.TempDir(), "results")
	testCacheDir := filepath.Join(t.TempDir(), "cache")
	os.MkdirAll(testResultsDir, 0755)
	os.MkdirAll(testCacheDir, 0755)

	args := models.ExecutionParams{
		Directories: []string{tempDir1, tempDir2, tempDir3, tempDir4},
		ResultsDir:  testResultsDir,
		CacheDir:    testCacheDir,
		CPUs:        2,
		BufSize:     1024,
	}

	err = app.StartExecution(args)
	if err != nil {
		t.Fatalf("app failed: %v", err)
	}

	csvLines, err := readResultsFile(t, testResultsDir)
	if err != nil {
		t.Fatal("Failed to read CSV data:", err)
	}
	// The three shared files must appear; dir3's unique file must not
	csvContainsExpected(t, csvLines, []string{"common.bin", "common_again.bin", "common_yet_again.bin"})
	for _, line := range csvLines {
		for _, field := range line {
			if field == "unique3.txt" {
				t.Error("unique3.txt from dir3 (no duplicates) should not appear in results")
			}
		}
	}
}
