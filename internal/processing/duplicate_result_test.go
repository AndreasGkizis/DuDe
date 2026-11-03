package processing

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"reflect"
	"testing"
)

func TestDuplicateResultsDoNotShareNestedSlicesAcrossBoundaries(t *testing.T) {
	newResult := func() models.DuplicateResult {
		return models.DuplicateResult{
			Groups: []models.FileHash{{
				FilePath: "/original",
				DuplicatesFound: []models.FileHash{{
					FilePath:        "/duplicate",
					DuplicatesFound: []models.FileHash{{FilePath: "/nested-duplicate"}},
				}},
			}},
			Diagnostics: models.NewScanDiagnostics(models.DiagnosticStatusCompleted),
		}
	}
	for _, boundary := range []string{"collection", "application publication", "getter"} {
		t.Run(boundary, func(t *testing.T) {
			app := NewApp(reporting.NoOpReporter{})
			publisher := &executionResultPublisher{}
			original := newResult()
			publisher.PublishDuplicateResult(original)

			var mutable *models.DuplicateResult
			if boundary == "collection" {
				mutable = &original
			} else {
				publisher.PublishTo(app)
				if boundary == "application publication" {
					mutable = publisher.duplicate
				} else {
					mutable = app.GetResults()
				}
			}
			mutable.Groups[0].FilePath = "/changed-original"
			mutable.Groups[0].DuplicatesFound[0].FilePath = "/changed-duplicate"
			mutable.Groups[0].DuplicatesFound[0].DuplicatesFound[0].FilePath = "/changed-nested-duplicate"
			if boundary == "collection" {
				publisher.PublishTo(app)
			}

			want := newResult()
			if got := app.GetResults(); !reflect.DeepEqual(got, &want) {
				t.Fatalf("mutation crossed %s boundary: got %#v, want %#v", boundary, got, &want)
			}
		})
	}
}

func TestCopyDuplicateResultPreservesNilAndEmptySlices(t *testing.T) {
	if copyDuplicateResult(nil) != nil {
		t.Fatal("copying a nil result should return nil")
	}
	for _, scenario := range []struct {
		name   string
		groups []models.FileHash
	}{
		{"nil groups", nil},
		{"empty groups", []models.FileHash{}},
		{"nil duplicates", []models.FileHash{{DuplicatesFound: nil}}},
		{"empty duplicates", []models.FileHash{{DuplicatesFound: []models.FileHash{}}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			source := models.DuplicateResult{
				Groups: scenario.groups, Diagnostics: models.NewScanDiagnostics(models.DiagnosticStatusCompleted),
			}
			if got := copyDuplicateResult(&source); !reflect.DeepEqual(got, &source) {
				t.Fatalf("copy changed slice representation: got %#v, want %#v", got, &source)
			}
		})
	}
}
