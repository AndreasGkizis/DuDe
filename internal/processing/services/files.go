package services

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	"strings"
	"time"

	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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

// CoverageReportRow is one deterministic row in a coverage report.
type CoverageReportRow struct {
	Type     string
	Scope    string
	Filename string
	FullPath string
	Size     string
	Reason   string
}

// SaveCoverageReport writes exactly one report for a completed coverage run.
func SaveCoverageReport(result models.MissingContentResult, diagnostics models.ScanDiagnostics, directory string) (string, error) {
	return saveCoverageReport(result, diagnostics, directory, func(path string) (io.WriteCloser, error) {
		return os.Create(path)
	}, os.Remove)
}

func saveCoverageReport(result models.MissingContentResult, diagnostics models.ScanDiagnostics, directory string, create func(string) (io.WriteCloser, error), remove func(string) error) (string, error) {
	filename := fmt.Sprintf("coverage_report%s.%s", time.Now().Format("_2006_01_02_15_04_05"), common.Results_file_extension)
	path := filepath.Join(directory, filename)
	file, err := create(path)
	if err != nil {
		return "", fmt.Errorf("create coverage report: %w", err)
	}

	if err := WriteCoverageCSV(file, result, diagnostics); err != nil {
		closeErr := file.Close()
		removeErr := remove(path)
		return "", errors.Join(err, closeErr, removeErr)
	}
	if err := file.Close(); err != nil {
		removeErr := remove(path)
		return "", errors.Join(fmt.Errorf("close coverage report: %w", err), removeErr)
	}
	return path, nil
}

// WriteCoverageCSV writes the coverage report header and deterministic rows.
func WriteCoverageCSV(output io.Writer, result models.MissingContentResult, diagnostics models.ScanDiagnostics) error {
	if _, err := output.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("failed to write coverage UTF-8 BOM: %w", err)
	}

	rows := make([]CoverageReportRow, 0, len(result.MissingFiles)+len(diagnostics.SkippedSourcePaths)+len(diagnostics.SkippedTargetPaths))
	for _, missing := range result.MissingFiles {
		rows = append(rows, CoverageReportRow{
			Type:     "Missing",
			Scope:    "Source",
			Filename: missing.Filename,
			FullPath: missing.FullPath,
			Size:     strconv.FormatInt(missing.Size, 10),
			Reason:   "Not found in target",
		})
	}
	for _, skipped := range diagnostics.SkippedSourcePaths {
		rows = append(rows, CoverageReportRow{Type: "Skipped", Scope: "Source", FullPath: skipped.Path, Reason: skipped.Reason})
	}
	for _, skipped := range diagnostics.SkippedTargetPaths {
		rows = append(rows, CoverageReportRow{Type: "Skipped", Scope: "Target", FullPath: skipped.Path, Reason: skipped.Reason})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Scope != rows[j].Scope {
			return rows[i].Scope < rows[j].Scope
		}
		if rows[i].FullPath != rows[j].FullPath {
			return rows[i].FullPath < rows[j].FullPath
		}
		return rows[i].Type < rows[j].Type
	})

	writer := csv.NewWriter(output)
	writer.Comma = GetDelimiterForOS()
	if err := writer.Write([]string{"Type", "Scope", "Filename", "Full Path", "Size (bytes)", "Reason"}); err != nil {
		return fmt.Errorf("failed to write coverage header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Type, row.Scope, row.Filename, row.FullPath, row.Size, row.Reason}); err != nil {
			return fmt.Errorf("failed to write coverage row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush coverage report: %w", err)
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
