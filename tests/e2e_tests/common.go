package e2e_tests

import (
	"DuDe/internal/common"
	process "DuDe/internal/processing"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var baseDir string = "./test_files/"

// FileType represents the type of test file to create
type FileType int

const (
	TextFile FileType = iota
	ImageFile
	AudioFile
	GreekFile
	MixedFile
)

// FileOptions contains options for test file creation
type FileOptions struct {
	// Number of files to create with duplicates
	DuplicateFileCount int
	// Number of files to create without duplicates
	UniqueFileCount int
	// Number of duplicates to create for each duplicate file (1 means one duplicate, resulting in 2 identical files)
	DuplicatesPerFile int
	// Types of files to create
	FileTypes []FileType
	// Size of files in bytes (for random content)
	FileSize int
	// Whether to create no-access files
	CreateNoAccessFiles bool
	// Prefix for file names
	Prefix string
}

// DefaultFileOptions returns default options for file creation
func DefaultFileOptions() FileOptions {
	return FileOptions{
		DuplicateFileCount: 3,
		UniqueFileCount:    2,
		DuplicatesPerFile:  1,
		FileTypes:          []FileType{TextFile, ImageFile, AudioFile},
		FileSize:           1024,
		CreateNoAccessFiles: false,
		Prefix:             "test",
	}
}

// buildBinary builds the Go application binary to be tested.
// It returns the binary path, temp directory, cleanup func.
func buildBinary(t *testing.T) (string, string, func()) {
	t.Helper()

	// Get the absolute path to the project root
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to get project root path: %v", err)
	}

	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	tempDir, err := os.MkdirTemp(baseDir, "dude-test-bin-")
	if err != nil {
		t.Fatalf("failed to create temp dir for binary: %v", err)
	}

	// Ensure tempDir is absolute
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}

	binaryName := "dude"
	if os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)

	// Use absolute path for the main package
	mainPkgPath := filepath.Join(projectRoot, "cmd", "main.go")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, mainPkgPath)
	cmd.Dir = projectRoot // Set working directory to project root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v\nCommand: %s", err, cmd.String())
	}

	// On Windows, we need to ensure the binary has the .exe extension for execution
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try with .exe if not found (for Windows)
		if !strings.HasSuffix(binaryPath, ".exe") {
			binaryPath += ".exe"
		}
	}

	// Verify the binary exists and is executable
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not found at %s: %v", binaryPath, err)
	}

	return binaryPath, tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// createTestFilesByteArray creates a temporary directory and populates it with the
// specified files and their content for testing.
// It returns the path to the created temporary directory and a cleanup
// function to remove the directory and its contents.
func createTestFilesByteArray(t *testing.T, files map[string][]byte) (string, func()) {
	t.Helper()
	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	// Create temp directory under the base directory
	tempDir, err := os.MkdirTemp(baseDir, "dude-test-data-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Ensure the path is in the correct format for the current OS
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}

	for path, content := range files {
		// Clean the path to handle any path separators correctly for the current OS
		path = filepath.Clean(path)
		fullPath := filepath.Join(tempDir, path)

		// Ensure the directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			cleanup()
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Create the file with the specified content
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			cleanup()
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	return tempDir, cleanup

}

// Reads and parses the CSV results file from the specified directory.
// It returns the parsed CSV lines or an error if any operation fails.
func readResultsFile(t *testing.T, dir string) ([][]string, error) {
	t.Helper()
	filePath := filepath.Join(dir, common.ResFilename)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to open results file %q: %w", filePath, err)
	}

	defer file.Close()

	bla := csv.NewReader(file)

	bla.Comma = process.GetDelimiterForOS()
	allCsvLines, err := bla.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data from %q: %w", filePath, err)
	}
	return allCsvLines, nil
}

// csvContainsExpected iterates through each expected word and verifies its existence within any of the CSV lines.
// If an expected word is not found, it reports a test error.
func csvContainsExpected(t *testing.T, allCsvLines [][]string, expectedWords []string) {
	t.Helper()
	for _, expectedWord := range expectedWords {
		found := false
		for _, line := range allCsvLines {
			if slices.Contains(line, expectedWord) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("File does not contain :\n%q", expectedWord)
		}
	}
}

// createTestFiles creates a temporary directory structure with test files based on provided options.
// It returns the path to the created temporary directory and a cleanup function.
func createTestFiles(t *testing.T, options FileOptions) (string, func()) {
	t.Helper()

	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	// Create temp directory under the base directory
	tempDir, err := os.MkdirTemp(baseDir, "dude-test-data-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Ensure the path is absolute
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}

	// Create subdirectories
	dirs := []string{"text_files", "image_files", "audio_files", "greek_files"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("failed to create directory %s: %v", dirPath, err)
		}
	}

	// Create files based on options
	for _, fileType := range options.FileTypes {
		switch fileType {
		case TextFile:
			createTextTestFiles(t, tempDir, options)
		case ImageFile:
			createImageTestFiles(t, tempDir, options)
		case AudioFile:
			createAudioTestFiles(t, tempDir, options)
		case GreekFile:
			createGreekTestFiles(t, tempDir, options)
		case MixedFile:
			createMixedTestFiles(t, tempDir, options)
		}
	}

	// Create no-access files if requested
	if options.CreateNoAccessFiles {
		createNoAccessFiles(t, tempDir, options)
	}

	cleanup := func() {
		// Try to restore access to no-access files before removal
		if options.CreateNoAccessFiles {
			noAccessPattern := filepath.Join(tempDir, "*no_access*")
			files, _ := filepath.Glob(noAccessPattern)
			for _, file := range files {
				os.Chmod(file, 0644)
			}
		}
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}

	return tempDir, cleanup
}

// createRandomContent generates random content of specified size
func createRandomContent(size int) ([]byte, error) {
	data := make([]byte, size)
	_, err := rand.Read(data)
	return data, err
}

// createFileWithContent creates a file with the given content
func createFileWithContent(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}

	return nil
}

// createTextTestFiles creates text files based on the provided options
func createTextTestFiles(t *testing.T, baseDir string, options FileOptions) {
	textDir := filepath.Join(baseDir, "text_files")

	// Create duplicate text files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Generate random name component
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-text-dup-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(textDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("%s-text-dup-%d-%d.txt", options.Prefix, r.Int64(), j)
			dupFilePath := filepath.Join(textDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique text files
	for i := 0; i < options.UniqueFileCount; i++ {
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-text-unique-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(textDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}
}

// createImageTestFiles creates image files based on the provided options
func createImageTestFiles(t *testing.T, baseDir string, options FileOptions) {
	imageDir := filepath.Join(baseDir, "image_files")

	// Image extensions to use
	imageExts := []string{".jpg", ".png", ".gif"}
	
	// Create duplicate image files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(imageExts))))
		ext := imageExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-img-dup-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(imageDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("%s-img-dup-%d-%d%s", options.Prefix, r.Int64(), j, ext)
			dupFilePath := filepath.Join(imageDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique image files
	for i := 0; i < options.UniqueFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(imageExts))))
		ext := imageExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-img-unique-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(imageDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}
}

// createAudioTestFiles creates audio files based on the provided options
func createAudioTestFiles(t *testing.T, baseDir string, options FileOptions) {
	audioDir := filepath.Join(baseDir, "audio_files")

	// Audio extensions to use
	audioExts := []string{".mp3", ".wav", ".ogg"}
	
	// Create duplicate audio files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(audioExts))))
		ext := audioExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-audio-dup-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(audioDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("%s-audio-dup-%d-%d%s", options.Prefix, r.Int64(), j, ext)
			dupFilePath := filepath.Join(audioDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique audio files
	for i := 0; i < options.UniqueFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(audioExts))))
		ext := audioExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-audio-unique-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(audioDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}
}

// createGreekTestFiles creates files with Greek characters in their names
func createGreekTestFiles(t *testing.T, baseDir string, options FileOptions) {
	greekDir := filepath.Join(baseDir, "greek_files")
	
	// Create duplicate Greek files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Generate random name component
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-έχει-αντίγραφο-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(greekDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("%s-έχει-αντίγραφο-%d-%d.txt", options.Prefix, r.Int64(), j)
			dupFilePath := filepath.Join(greekDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique Greek files
	for i := 0; i < options.UniqueFileCount; i++ {
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-δέν-έχει-αντίγραφο-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(greekDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}
}

// createMixedTestFiles creates files in the base directory with mixed types
func createMixedTestFiles(t *testing.T, baseDir string, options FileOptions) {
	// Mixed extensions to use
	mixedExts := []string{".pdf", ".zip", ".doc", ".xls", ".csv"}
	
	// Create duplicate mixed files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(mixedExts))))
		ext := mixedExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-mixed-dup-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(baseDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("%s-mixed-dup-%d-%d%s", options.Prefix, r.Int64(), j, ext)
			dupFilePath := filepath.Join(baseDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique mixed files
	for i := 0; i < options.UniqueFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(mixedExts))))
		ext := mixedExts[r.Int64()]
		
		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-mixed-unique-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(baseDir, fileName)

		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
	}
}

// createNoAccessFiles creates files with no access permissions
func createNoAccessFiles(t *testing.T, baseDir string, options FileOptions) {
	// Create files with no access permissions
	for i := 0; i < options.DuplicateFileCount; i++ {
		fileName := fmt.Sprintf("%s-no_access_file-%d.txt", options.Prefix, i)
		filePath := filepath.Join(baseDir, fileName)
		
		// Create random content
		content, err := createRandomContent(options.FileSize)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}
		
		// Remove all permissions
		if err := os.Chmod(filePath, 0); err != nil {
			t.Fatalf("failed to set no permissions for file %s: %v", filePath, err)
		}
	}
}
