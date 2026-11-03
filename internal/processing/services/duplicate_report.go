package services

import (
	"DuDe/internal/common"
	"DuDe/internal/models"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SaveDuplicateReport writes duplicate and skipped diagnostic rows. It returns
// an empty path when there are no rows to write.
func SaveDuplicateReport(data *sync.Map, diagnostics models.ScanDiagnostics, directory string) (string, error) {
	return saveDuplicateReport(data, diagnostics, directory, func(path string) (io.WriteCloser, error) {
		return os.Create(path)
	}, os.Remove)
}

func saveDuplicateReport(data *sync.Map, diagnostics models.ScanDiagnostics, directory string, create func(string) (io.WriteCloser, error), remove func(string) error) (string, error) {
	entries := GetFlattened(data)
	if len(entries) == 0 && len(diagnostics.Diagnostics) == 0 {
		return "", nil
	}

	path := nextAvailableReportPath(directory, "duplicate_results")
	file, err := create(path)
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

func nextAvailableReportPath(directory, prefix string) string {
	timestamp := time.Now().Format("_2006_01_02_15_04_05")
	suffix := 0
	for {
		name := fmt.Sprintf("%s%s.%s", prefix, timestamp, common.Results_file_extension)
		if suffix > 0 {
			name = fmt.Sprintf("%s%s_%d.%s", prefix, timestamp, suffix, common.Results_file_extension)
		}
		path := filepath.Join(directory, name)
		if _, err := os.Stat(path); err != nil {
			return path
		}
		suffix++
	}
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
