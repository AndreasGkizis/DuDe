package unit_tests

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"

	"DuDe/internal/models"
	"DuDe/internal/processing/services"
)

func TestWriteCoverageCSVWritesDeterministicMissingAndSkippedRows(t *testing.T) {
	var output bytes.Buffer
	result := models.MissingContentResult{MissingFiles: []models.MissingSourceFile{
		{Filename: "missing.txt", FullPath: "/source/missing.txt", Size: 12},
	}}
	diagnostics := models.ScanDiagnostics{
		Status: models.DiagnosticStatusCompleted,
		Diagnostics: []models.Diagnostic{
			{Path: "/source/private", Scope: models.DiagnosticScopeSource, Phase: models.DiagnosticPhaseWalking, Reason: "permission denied"},
			{Path: "/target/private", Scope: models.DiagnosticScopeTarget, Phase: models.DiagnosticPhaseWalking, Reason: "permission denied"},
		},
	}

	if err := services.WriteCoverageCSV(&output, result, diagnostics); err != nil {
		t.Fatalf("write coverage CSV: %v", err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("expected UTF-8 BOM")
	}

	reader := csv.NewReader(strings.NewReader(output.String()[3:]))
	reader.Comma = services.GetDelimiterForOS()
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read coverage CSV: %v", err)
	}
	want := [][]string{
		{"Type", "Scope", "Filename", "Full Path", "Size (bytes)", "Reason"},
		{"Missing", "source", "missing.txt", "/source/missing.txt", "12", "Not found in target"},
		{"Skipped", "source", "", "/source/private", "", "permission denied (phase: walking)"},
		{"Skipped", "target", "", "/target/private", "", "permission denied (phase: walking)"},
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
	if got := strings.Count(output.String(), "Type"); got != 1 {
		t.Fatalf("expected one header row, got %d occurrences", got)
	}
}
