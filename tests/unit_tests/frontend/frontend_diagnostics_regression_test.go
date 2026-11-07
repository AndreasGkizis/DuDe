package unit_tests

import (
	"os"
	"strings"
	"testing"
)

func TestDuplicateDiagnosticsHaveVisibleContainerOutsideCoverageSummary(t *testing.T) {
	content, err := os.ReadFile("../../../frontend/src/template.html")
	if err != nil {
		t.Fatalf("read frontend template: %v", err)
	}
	template := string(content)
	coverageStart := strings.Index(template, `<div id="coverage-summary"`)
	coverageEnd := strings.Index(template[coverageStart:], `<!-- OPEN RESULTS FOLDER`)
	if coverageStart < 0 || coverageEnd < 0 {
		t.Fatal("could not locate coverage summary boundaries")
	}
	coverageEnd += coverageStart
	diagnosticsIndex := strings.Index(template, `id="coverage-diagnostics"`)
	if diagnosticsIndex < 0 {
		t.Fatal("diagnostic container is missing")
	}
	if diagnosticsIndex > coverageStart && diagnosticsIndex < coverageEnd {
		t.Fatal("diagnostic container is nested inside the coverage-only summary")
	}
}
