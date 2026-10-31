package services

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	"strings"
	"time"

	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func SaveResultsAsCSV(data *sync.Map, fulldir string) error {
	flattened_data := GetFlattened(data)
	log.InfoWithFuncName(fmt.Sprintf("Number of duplicates found: %d", len(flattened_data)))
	log.InfoWithFuncName(fmt.Sprintf("Creating results file in: %s", fulldir))

	if len(flattened_data) == 0 {
		log.WarnWithFuncName("No results file produced, 0 duplicates found")
		return nil
	}

	file, err := createResultFile(fulldir)
	if err != nil {
		return err
	}

	if err := WriteResultsCSV(file, flattened_data); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return fmt.Errorf("failed to close results file: %w", err)
	}

	return nil
}

func WriteResultsCSV(output io.Writer, entries []models.ResultEntry) error {
	// Write the UTF-8 BOM bytes at the very beginning of the file to force stupid excel to recognise the encoding.
	if _, err := output.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("failed to write UTF-8 BOM: %w", err)
	}

	writer := csv.NewWriter(output)
	writer.Comma = GetDelimiterForOS()

	if err := writer.Write(common.ResultsHeader); err != nil {
		return fmt.Errorf("failed to write results header: %w", err)
	}

	for _, entry := range entries {
		if err := writer.Write([]string{
			entry.Filename,
			entry.FullPath,
			entry.DuplicateFilename,
			entry.DuplicateFullPath,
		}); err != nil {
			return fmt.Errorf("failed to write results row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush results file: %w", err)
	}

	return nil
}

func ResultsFileExist(path string) bool {

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return false
	}

	for _, entry := range entries {
		// Skip directories to ensure we only process files
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if file starts with "results" and ends with ".csv"
		if strings.HasPrefix(name, "results") && strings.HasSuffix(name, ".csv") {
			fmt.Printf("Found matching file: %s\n", name)
			return true
		}
	}
	return false
}

func GetDelimiterForOS() rune {
	var delimiter rune
	if runtime.GOOS == "windows" {
		delimiter = ';'
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	} else {
		delimiter = ',' // Default for Linux, macOS, etc.
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	}
	return delimiter
}

func createResultFile(fulldir string) (*os.File, error) {

	datetime := time.Now()
	filename := fmt.Sprint(common.Results_file_name, datetime.Format("_2006_01_02_15_04_05"), ".", common.Results_file_extension)
	filepath := filepath.Join(fulldir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Results file with name [%s] already exists", filename))
		return nil, err
	}
	return file, nil
}
