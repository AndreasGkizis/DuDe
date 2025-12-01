package e2e_tests

import (
	"DuDe/internal/common"
	"DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	process "DuDe/internal/processing"
	"DuDe/internal/reporting"
	"context"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
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
var dataSubDir string = "dude-test-data-"
var binSubDir string = "dude-test-bin-"

// FileType represents the type of test file to create
type FileType int

const (
	TextFile FileType = iota
	ImageFile
	AudioFile
)

// FileOptions contains options for test file creation
type FileOptions struct {
	// Number of files to create with duplicates
	DuplicateFileCount int
	// Number of duplicates to create for each duplicate file (1 means one duplicate, resulting in 2 identical files)
	DuplicatesPerFile int
	// Number of files to create without duplicates
	UniqueFileCount int
	// Types of files to create
	FileTypes []FileType
	// Whether to create no-access files
	CreateNoAccessFiles bool
	// Prefix for file names
	Prefix string
}

func (fo FileOptions) CalculateTotalDuplicateFiles() int {
	// Ensure FileTypes slice is not nil to avoid panics when accessing len().
	// If FileTypes is nil, its length is considered 0 for the calculation.
	fileTypesCount := 0
	if fo.FileTypes != nil {
		fileTypesCount = len(fo.FileTypes)
	}

	// Perform the calculation.
	// Cast to int64 to prevent potential overflow if intermediate results are large.
	totalDuplicateFiles := fo.DuplicateFileCount * fo.DuplicatesPerFile * fileTypesCount

	return totalDuplicateFiles
}

// DefaultFileOptions returns default options for file creation
func DefaultFileOptions() FileOptions {
	return FileOptions{
		DuplicateFileCount:  3,
		UniqueFileCount:     2,
		DuplicatesPerFile:   1,
		FileTypes:           []FileType{TextFile, ImageFile, AudioFile},
		CreateNoAccessFiles: false,
		Prefix:              "test",
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

	tempDir, err := os.MkdirTemp(baseDir, binSubDir)
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

// csvContainsNumberOfRowsExpected counts the number of data rows in the provided CSV content.
// A row is considered a data row if it does not contain the common.ResultsFileSeparator,
// and it is not a header row as identified by common.ResultsHeader.
// It reports a test error if the counted number of data rows does not match the expectedRows.
func csvContainsNumberOfRowsExpected(t *testing.T, allCsvLines [][]string, expectedRows int) {
	t.Helper()
	found := 0
	for _, line := range allCsvLines {

		if !slices.Contains(line, common.ResultsFileSeperator) &&
			!slices.Equal(line[len(line)-3:], common.ResultsHeader[len(common.ResultsHeader)-3:]) {
			found++
		}
	}
	if found != expectedRows {
		t.Errorf("Expected %d data rows, but found %d. Full CSV content: %v", expectedRows, found, allCsvLines)
	}
}

// createTestFiles creates a temporary directory structure with test files based on provided options.
// It returns the path to the created temporary directory and a cleanup function.
func createTestFiles(t *testing.T, options FileOptions) (string, func()) {
	t.Helper()

	tempDir, _ := createBaseDirs(t)

	// Create files based on options
	for _, fileType := range options.FileTypes {
		switch fileType {
		case TextFile:
			createTextTestFiles(t, tempDir, options)
		case ImageFile:
			createImageTestFiles(t, tempDir, options)
		case AudioFile:
			createAudioTestFiles(t, tempDir, options)
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

func createBaseDirs(t *testing.T) (string, error) {
	// Ensure baseDir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base directory: %v", err)
	}

	// Create temp directory under the base directory
	tempDir, err := os.MkdirTemp(baseDir, dataSubDir)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Ensure the path is absolute
	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get absolute path for temp dir: %v", err)
	}
	return tempDir, nil
}

// createRandomContent generates random content of specified size
func createRandomContent(size int) ([]byte, error) {
	data := make([]byte, size)
	_, err := rand.Read(data)
	return data, err
}

func createRandomText(length int) ([]byte, error) {
	asciiPrintable := generateAllEnglishAndGreekLettersAndSymbols()
	charSetLength := big.NewInt(int64(len(asciiPrintable)))
	bytes := make([]byte, length)

	for i := range bytes {
		randomIndex, _ := rand.Int(rand.Reader, charSetLength)
		bytes[i] = asciiPrintable[randomIndex.Int64()]
	}

	return bytes, nil
}

func generateAllEnglishAndGreekLettersAndSymbols() string {
	var sb strings.Builder

	// 1. English Letters (ASCII - Basic Latin Block)
	// Uppercase: A-Z (U+0041 to U+005A)
	for i := 0x0041; i <= 0x005A; i++ {
		sb.WriteRune(rune(i))
	}
	// Lowercase: a-z (U+0061 to U+007A)
	for i := 0x0061; i <= 0x007A; i++ {
		sb.WriteRune(rune(i))
	}

	// 2. Greek Letters (Unicode - Greek and Coptic Block)
	// Uppercase: Alpha to Omega (U+0391 to U+03A9) - excludes some archaic/variant forms
	for i := 0x0391; i <= 0x03A9; i++ {
		sb.WriteRune(rune(i))
	}
	// Lowercase: alpha to omega (U+03B1 to Ux03C9) - excludes some archaic/variant forms
	for i := 0x03B1; i <= 0x03C9; i++ {
		sb.WriteRune(rune(i))
	}

	// For 'ό', the combining acute accent is U+0301.
	for i := 0x0300; i <= 0x036F; i++ {
		// Filter out some less common or control-like combining marks if desired,
		// but for a "wider charset," including the whole range is fine.
		sb.WriteRune(rune(i))
	}
	// 3. Common Symbols
	// These are scattered across various Unicode blocks.
	// We'll primarily focus on common ASCII symbols and some general punctuation/math symbols.

	// ASCII Printable Symbols (U+0021 to U+002F, U+0030 to U+0039 (digits), U+003A to U+0040, U+005B to U+0060, U+007B to U+007E)
	asciiSymbolsAndDigits := "!\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~"
	sb.WriteString(asciiSymbolsAndDigits)

	// Add some common general punctuation and mathematical symbols from other blocks
	// This is a curated list, not an exhaustive range, as "all symbols" is extremely broad.
	sb.WriteString("€£¥§©®™℗℅™№¶•—–…“”‘’†‡⁂‰‱′″‴‟")      // General Punctuation, Currency Symbols
	sb.WriteString("±×÷¼½¾¹²³⁴⁵⁶⁷⁸⁹⁰∞µΩ∆∏∑√∫≅≈≠≡≤≥∂∇⊕⊗") // Mathematical Operators, Superscripts
	sb.WriteString("←↑→↓↔↕↖↗↘↙♠♣♥♦")                     // Arrows, Geometric Shapes, Dingbats (e.g., card suits)

	return sb.String()
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
	currentDir := filepath.Join(baseDir, "text_files")
	fileLength := 100

	if err := os.MkdirAll(currentDir, os.ModePerm); err != nil {
		// os.RemoveAll(tempDir)
		t.Fatalf("failed to create directory %s: %v", currentDir, err)
	}

	// Create duplicate text files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Generate random name component
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-text-hasdup-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(currentDir, fileName)

		// Create random content
		content, err := createRandomText(fileLength)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("isdup-%d-%s", j, fileName)
			dupFilePath := filepath.Join(currentDir, dupFileName)
			if err := copyFile(filePath, dupFilePath); err != nil {
				t.Fatalf("failed to copy file: %v", err)
			}
		}
	}

	// Create unique text files
	for i := 0; i < options.UniqueFileCount; i++ {
		r, _ := rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-text-unique-%d.txt", options.Prefix, r.Int64())
		filePath := filepath.Join(currentDir, fileName)

		// Create random content
		content, err := createRandomText(fileLength)
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
	currentDir := filepath.Join(baseDir, "image_files")
	imageExts := []string{".jpg", ".png", ".gif"}

	if err := os.MkdirAll(currentDir, os.ModePerm); err != nil {
		// os.RemoveAll(tempDir)
		t.Fatalf("failed to create directory %s: %v", currentDir, err)
	}
	// Image extensions to use

	// Create duplicate image files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(imageExts))))
		ext := imageExts[r.Int64()]

		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-img-hasdup-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(currentDir, fileName)
		createRandomImage(filePath, "png")

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("isdup-%d-%s", j, fileName)
			dupFilePath := filepath.Join(currentDir, dupFileName)
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
		filePath := filepath.Join(currentDir, fileName)

		createRandomImage(filePath, "png")
	}
}

// createRandomImage generates an image filled with random colors and saves it
// to the specified output file in the given format.
//
// Parameters:
//
//	width: The width of the image in pixels.
//	height: The height of the image in pixels.
//	outputFilename: The full path and name of the output file (e.g., "my_image.png").
//	imageType: The desired image format (e.g., "png", "jpeg", "gif").
//
// Returns:
//
//	An error if image generation or encoding fails, otherwise nil.
func createRandomImage(outputFilename, imageType string) error {
	width, height := 256, 256
	// Create a new random number generator using the current time as a seed.
	// This ensures a different random pattern each time the program runs.
	r, _ := rand.Int(rand.Reader, big.NewInt(256))

	// Create a new RGBA image. RGBA supports Red, Green, Blue, and Alpha channels.
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill the image with completely random colors
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Generate random values for Red, Green, and Blue channels (0-255)
			randomRed := uint8(r.Int64())
			randomGreen := uint8(r.Int64())
			randomBlue := uint8(r.Int64())

			// Set Alpha to 255 for full opacity (no transparency)
			alpha := uint8(255)

			// Create an RGBA color and set the pixel
			img.SetRGBA(x, y, color.RGBA{R: randomRed, G: randomGreen, B: randomBlue, A: alpha})
		}
	}

	// Create the output file
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFilename, err)
	}
	defer outputFile.Close() // Ensure the file is closed

	// Encode the image based on the specified type
	// Convert imageType to lowercase for case-insensitive comparison
	switch strings.ToLower(imageType) {
	case "png":
		err = png.Encode(outputFile, img)
	case "jpeg", "jpg": // Handle both "jpeg" and "jpg" extensions
		// JPEG encoding options can be set here. Quality ranges from 1 to 100.
		// A quality of 75 is a common default.
		err = jpeg.Encode(outputFile, img, &jpeg.Options{Quality: 90})
	case "gif":
		// GIF encoding options. The delay is for animation, but for a single frame,
		// it's not strictly necessary. We just need to put the image in a GIF struct.
		err = gif.Encode(outputFile, img, nil) // nil for default options
	default:
		return fmt.Errorf("unsupported image type: %s. Supported types are png, jpeg/jpg, gif", imageType)
	}

	if err != nil {
		return fmt.Errorf("failed to encode image as %s: %w", imageType, err)
	}

	log.Printf("Successfully generated %s as %s.", outputFilename, imageType)
	return nil // No error
}

// createAudioTestFiles creates audio files based on the provided options
func createAudioTestFiles(t *testing.T, baseDir string, options FileOptions) {
	currentDir := filepath.Join(baseDir, "audio_files")
	audioExts := []string{".mp3", ".wav", ".ogg"}
	size := 100

	if err := os.MkdirAll(currentDir, os.ModePerm); err != nil {
		// os.RemoveAll(tempDir)
		t.Fatalf("failed to create directory %s: %v", currentDir, err)
	}

	// Create duplicate audio files
	for i := 0; i < options.DuplicateFileCount; i++ {
		// Choose random extension
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(len(audioExts))))
		ext := audioExts[r.Int64()]

		// Generate random name component
		r, _ = rand.Int(rand.Reader, big.NewInt(1000))
		fileName := fmt.Sprintf("%s-audio-hasdup-%d%s", options.Prefix, r.Int64(), ext)
		filePath := filepath.Join(currentDir, fileName)

		// Create random content
		content, err := createRandomContent(size)
		if err != nil {
			t.Fatalf("failed to create random content: %v", err)
		}

		// Write original file
		if err := createFileWithContent(filePath, content); err != nil {
			t.Fatalf("failed to create file %s: %v", filePath, err)
		}

		// Create duplicates
		for j := 1; j <= options.DuplicatesPerFile; j++ {
			dupFileName := fmt.Sprintf("isdup-%d-%s", j, fileName)
			dupFilePath := filepath.Join(currentDir, dupFileName)
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
		filePath := filepath.Join(currentDir, fileName)

		// Create random content
		content, err := createRandomContent(size)
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

	size := 100

	// Create files with no access permissions
	for i := 0; i < options.DuplicateFileCount; i++ {
		fileName := fmt.Sprintf("%s-no_access_file-%d.txt", options.Prefix, i)
		filePath := filepath.Join(baseDir, fileName)

		// Create random content
		content, err := createRandomContent(size)
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

// llm slop

// setupTestApp creates an app instance with real dependencies for E2E testing
func setupTestApp(t *testing.T) *process.FrontendApp {

	wailsReporter := reporting.NoOpReporter{}

	app := process.NewApp(&wailsReporter)
	app.Startup(context.Background()) // Initialize context (required by Wails structure)

	return app
}

// executeE2E runs the full validation and execution flow.
func executeE2E(t *testing.T, app *process.FrontendApp, resolver validation.Resolver, args *models.ExecutionParams) error {
	// 1. Validation (Same as your app.StartExecution wrapper)
	// NOTE: Need to find a way to get the executableDir if defaults are used.
	// For E2E, we can use the TempDir() as the executableDir fallback.
	exeDir := filepath.Join(t.TempDir(), "exe_mock")

	if err := resolver.ResolveAndValidateArgs(args, exeDir); err != nil {
		return fmt.Errorf("E2E Validation Failed: %w", err)
	}

	// 2. Execution
	app.Args = *args
	// Call the internal execution function directly (requires being in the processing_test package)
	// return process.StartExecution(app)
	return nil
}
