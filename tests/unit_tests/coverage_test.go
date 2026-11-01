package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMatchCoverageMatchesByContentMetadataNotPath(t *testing.T) {
	source := []models.FileHash{
		{FileName: "old.txt", FilePath: "/source/old.txt", FileSize: 3, Hash: "same"},
	}
	target := []models.FileHash{
		{FileName: "renamed.txt", FilePath: "/target/nested/renamed.txt", FileSize: 3, Hash: "same"},
	}

	result, err := services.MatchCoverage(context.Background(), source, target, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if result.TotalSourceFiles != 1 || len(result.MissingFiles) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestMatchCoverageReportsEveryUncoveredSourceAndSortsPaths(t *testing.T) {
	source := []models.FileHash{
		{FileName: "z.txt", FilePath: "/source/z.txt", FileSize: 1, Hash: "z"},
		{FileName: "a.txt", FilePath: "/source/a.txt", FileSize: 2, Hash: "a"},
	}

	result, err := services.MatchCoverage(context.Background(), source, nil, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	want := []models.MissingSourceFile{
		{Filename: "a.txt", FullPath: "/source/a.txt", Size: 2},
		{Filename: "z.txt", FullPath: "/source/z.txt", Size: 1},
	}
	if !reflect.DeepEqual(result.MissingFiles, want) {
		t.Fatalf("missing files = %#v, want %#v", result.MissingFiles, want)
	}
}

func TestMatchCoverageAllowsOneTargetToCoverMultipleSources(t *testing.T) {
	source := []models.FileHash{
		{FileName: "one.txt", FilePath: "/source/one.txt", FileSize: 4, Hash: "same"},
		{FileName: "two.txt", FilePath: "/source/one/two.txt", FileSize: 4, Hash: "same"},
	}
	target := []models.FileHash{
		{FileName: "copy.txt", FilePath: "/target/copy.txt", FileSize: 4, Hash: "same"},
	}

	result, err := services.MatchCoverage(context.Background(), source, target, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if len(result.MissingFiles) != 0 {
		t.Fatalf("expected both source files to be covered, got %#v", result.MissingFiles)
	}
}

func TestMatchCoverageDoesNotUseSourceOnlyDuplicates(t *testing.T) {
	source := []models.FileHash{
		{FileName: "one.txt", FilePath: "/source/one.txt", FileSize: 4, Hash: "same"},
		{FileName: "two.txt", FilePath: "/source/two.txt", FileSize: 4, Hash: "same"},
	}

	result, err := services.MatchCoverage(context.Background(), source, nil, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if len(result.MissingFiles) != len(source) {
		t.Fatalf("expected both source files to be missing, got %#v", result.MissingFiles)
	}
}

func TestMatchCoverageIgnoresTargetOnlyFilesAndSupportsEmptySource(t *testing.T) {
	target := []models.FileHash{{FileName: "extra", FilePath: "/target/extra", FileSize: 5, Hash: "extra"}}
	result, err := services.MatchCoverage(context.Background(), nil, target, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if result.TotalSourceFiles != 0 || len(result.MissingFiles) != 0 {
		t.Fatalf("unexpected empty-source result: %#v", result)
	}
}

func TestMatchCoverageHandlesZeroByteFiles(t *testing.T) {
	source := []models.FileHash{{FileName: "empty", FilePath: "/source/empty", FileSize: 0, Hash: "empty-hash"}}
	target := []models.FileHash{{FileName: "copy", FilePath: "/target/copy", FileSize: 0, Hash: "empty-hash"}}

	result, err := services.MatchCoverage(context.Background(), source, target, false)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if len(result.MissingFiles) != 0 {
		t.Fatalf("expected zero-byte file to be covered, got %#v", result.MissingFiles)
	}
}

func TestMatchCoverageParanoidModeRejectsHashCollision(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source.txt")
	targetPath := filepath.Join(root, "target.txt")
	if err := os.WriteFile(sourcePath, []byte("same-size"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("other!!!!"), 0o600); err != nil {
		t.Fatal(err)
	}

	source := []models.FileHash{{FileName: "source.txt", FilePath: sourcePath, FileSize: 9, Hash: "collision"}}
	target := []models.FileHash{{FileName: "target.txt", FilePath: targetPath, FileSize: 9, Hash: "collision"}}
	result, err := services.MatchCoverage(context.Background(), source, target, true)
	if err != nil {
		t.Fatalf("match coverage: %v", err)
	}
	if len(result.MissingFiles) != 1 {
		t.Fatalf("expected collision to be rejected, got %#v", result.MissingFiles)
	}
}

func TestMatchCoverageStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := services.MatchCoverage(ctx, []models.FileHash{{FilePath: "/source/file", FileSize: 1}}, nil, false)
	if err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
