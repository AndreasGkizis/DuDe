package services

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
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
	"time"
)

// createReportFile retries timestamped names with numeric suffixes on collision.
// create must use exclusive creation so existing reports are never overwritten.
func createReportFile(directory, prefix string, create func(string) (io.WriteCloser, error)) (string, io.WriteCloser, error) {
	timestamp := time.Now().Format("_2006_01_02_15_04_05")
	for suffix := 0; ; suffix++ {
		name := fmt.Sprintf("%s%s.%s", prefix, timestamp, common.Results_file_extension)
		if suffix > 0 {
			name = fmt.Sprintf("%s%s_%d.%s", prefix, timestamp, suffix, common.Results_file_extension)
		}
		path := filepath.Join(directory, name)
		file, err := create(path)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		return path, file, nil
	}
}

// SaveDuplicateReport writes duplicate and skipped diagnostic rows. It returns
// an empty path when there are no rows to write.
func SaveDuplicateReport(data *sync.Map, diagnostics models.ScanDiagnostics, directory string) (string, error) {
	return saveDuplicateReport(data, diagnostics, directory, func(path string) (io.WriteCloser, error) {
		return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	}, os.Remove)
}

func saveDuplicateReport(data *sync.Map, diagnostics models.ScanDiagnostics, directory string, create func(string) (io.WriteCloser, error), remove func(string) error) (string, error) {
	entries := GetFlattened(data)
	log.DebugWithFuncName(fmt.Sprintf("Number of duplicates found: %d", countDuplicateEntries(data)))
	if len(entries) == 0 && len(diagnostics.Diagnostics) == 0 {
		return "", nil
	}

	path, file, err := createReportFile(directory, "duplicate_results", create)
	if err != nil {
		return "", fmt.Errorf("create duplicate report: %w", err)
	}
	if err := writeDuplicateCSV(file, entries, diagnostics); err != nil {
		closeErr := file.Close()
		removeErr := remove(path)
		return "", errors.Join(err, closeErr, removeErr)
	}
	if err := file.Close(); err != nil {
		removeErr := remove(path)
		return "", errors.Join(fmt.Errorf("close duplicate report: %w", err), removeErr)
	}
	return path, nil
}

func countDuplicateEntries(data *sync.Map) (count int) {
	data.Range(func(_, value any) bool {
		if fileHash, ok := value.(models.FileHash); ok {
			count += len(fileHash.DuplicatesFound)
		}
		return true
	})
	return count
}

// duplicateReportRow is intentionally separate from coverage report rows.
type duplicateReportRow struct {
	Type              string
	Filename          string
	FullPath          string
	DuplicateFilename string
	DuplicateFullPath string
	Reason            string
}

func writeDuplicateCSV(output io.Writer, entries []models.ResultEntry, diagnostics models.ScanDiagnostics) error {
	rows := make([]duplicateReportRow, 0, len(entries)+len(diagnostics.Diagnostics))
	for _, entry := range entries {
		if entry.Filename == common.ResultsFileSeperator {
			continue
		}
		rows = append(rows, duplicateReportRow{
			Type:              "Duplicate",
			Filename:          entry.Filename,
			FullPath:          entry.FullPath,
			DuplicateFilename: entry.DuplicateFilename,
			DuplicateFullPath: entry.DuplicateFullPath,
			Reason:            "identical content",
		})
	}
	for _, diagnostic := range diagnostics.Diagnostics {
		rows = append(rows, duplicateReportRow{
			Type:     "Skipped",
			FullPath: diagnostic.Path,
			Reason:   diagnosticReason(diagnostic),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Type != rows[j].Type {
			return rows[i].Type < rows[j].Type
		}
		if rows[i].FullPath != rows[j].FullPath {
			return rows[i].FullPath < rows[j].FullPath
		}
		if rows[i].DuplicateFullPath != rows[j].DuplicateFullPath {
			return rows[i].DuplicateFullPath < rows[j].DuplicateFullPath
		}
		return rows[i].Reason < rows[j].Reason
	})

	if _, err := output.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("failed to write duplicate UTF-8 BOM: %w", err)
	}
	writer := csv.NewWriter(output)
	writer.Comma = GetDelimiterForOS()
	if err := writer.Write([]string{"Type", "Filename", "Full Path", "Duplicate Filename", "Duplicate Full Path", "Reason"}); err != nil {
		return fmt.Errorf("failed to write duplicate header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Type, row.Filename, row.FullPath, row.DuplicateFilename, row.DuplicateFullPath, row.Reason}); err != nil {
			return fmt.Errorf("failed to write duplicate row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush duplicate report: %w", err)
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
		return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	}, os.Remove)
}

func saveCoverageReport(result models.MissingContentResult, diagnostics models.ScanDiagnostics, directory string, create func(string) (io.WriteCloser, error), remove func(string) error) (string, error) {
	path, file, err := createReportFile(directory, "coverage_report", create)
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

	rows := make([]CoverageReportRow, 0, len(result.MissingFiles)+len(diagnostics.Diagnostics))
	for _, missing := range result.MissingFiles {
		rows = append(rows, CoverageReportRow{
			Type:     "Missing",
			Scope:    "source",
			Filename: missing.Filename,
			FullPath: missing.FullPath,
			Size:     strconv.FormatInt(missing.Size, 10),
			Reason:   "Not found in target",
		})
	}
	for _, diagnostic := range diagnostics.Diagnostics {
		rows = append(rows, CoverageReportRow{
			Type:     "Skipped",
			Scope:    string(diagnostic.Scope),
			FullPath: diagnostic.Path,
			Reason:   diagnosticReason(diagnostic),
		})
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

func diagnosticReason(diagnostic models.Diagnostic) string {
	if diagnostic.Phase == "" {
		return diagnostic.Reason
	}
	return fmt.Sprintf("%s (phase: %s)", diagnostic.Reason, diagnostic.Phase)
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
