package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

func TestWriteCoverageCSVWritesDeterministicMissingAndSkippedRows(t *testing.T) {
	var output bytes.Buffer
	result := models.MissingContentResult{MissingFiles: []models.MissingSourceFile{
		{Filename: "missing.txt", FullPath: "/source/missing.txt", Size: 12},
	}}
	diagnostics := models.ScanDiagnostics{
		SkippedSourcePaths: []models.SkippedPath{{Path: "/source/private", Reason: "permission denied"}},
		SkippedTargetPaths: []models.SkippedPath{{Path: "/target/private", Reason: "permission denied"}},
	}

	if err := services.WriteCoverageCSV(&output, result, diagnostics); err != nil {
		t.Fatalf("write coverage CSV: %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("expected UTF-8 BOM")
	}

	reader := csv.NewReader(strings.NewReader(string(output.Bytes()[3:])))
	reader.Comma = services.GetDelimiterForOS()
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read coverage CSV: %v", err)
	}
	want := [][]string{
		{"Type", "Scope", "Filename", "Full Path", "Size (bytes)", "Reason"},
		{"Missing", "Source", "missing.txt", "/source/missing.txt", "12", "Not found in target"},
		{"Skipped", "Source", "", "/source/private", "", "permission denied"},
		{"Skipped", "Target", "", "/target/private", "", "permission denied"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("coverage rows = %#v, want %#v", rows, want)
	}
}

func TestWriteCoverageCSVWritesHeaderForEmptyResult(t *testing.T) {
	var output bytes.Buffer
	if err := services.WriteCoverageCSV(&output, models.MissingContentResult{}, models.ScanDiagnostics{}); err != nil {
		t.Fatalf("write empty coverage CSV: %v", err)
	}
	if got := strings.Count(string(output.Bytes()), "Type"); got != 1 {
		t.Fatalf("expected one header row, got %d occurrences", got)
	}
}
