package unit_tests

import (
	"DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	"DuDe/internal/processing/flows"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuplicateFlowRequiresDirectories(t *testing.T) {
	flow := flows.NewDuplicateFlow(validation.MockValidator{})

	err := flow.Validate(&models.ExecutionParams{})

	if !errors.Is(err, flows.ErrNoDirectories) {
		t.Fatalf("expected ErrNoDirectories, got %v", err)
	}
}

func TestDuplicateFlowValidatesEveryDirectory(t *testing.T) {
	var paths []string
	flow := flows.NewDuplicateFlow(validation.MockValidator{
		ReadableDirFunc: func(path string) error {
			paths = append(paths, path)
			return nil
		},
	})
	args := models.ExecutionParams{Directories: []string{"/one", "/two"}}

	err := flow.Validate(&args)

	if err != nil {
		t.Fatalf("validate duplicate flow: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/one" || paths[1] != "/two" {
		t.Fatalf("validated paths = %v, want [/one /two]", paths)
	}
}

func TestDuplicateFlowWrapsDirectoryReadError(t *testing.T) {
	expected := validation.ErrNoReadAccess
	flow := flows.NewDuplicateFlow(validation.MockValidator{
		ReadableDirFunc: func(path string) error {
			if path == "/bad" {
				return expected
			}
			return nil
		},
	})
	args := models.ExecutionParams{Directories: []string{"/good", "/bad"}}

	err := flow.Validate(&args)

	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
	if !strings.HasPrefix(err.Error(), `Directories[1] ("/bad")`) {
		t.Fatalf("unexpected directory error: %v", err)
	}
}

func TestFindMissingContentFlowRequiresSourceAndTarget(t *testing.T) {
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{})

	err := flow.Validate(&models.ExecutionParams{})
	if !errors.Is(err, flows.ErrSourceDirectoryRequired) {
		t.Fatalf("expected ErrSourceDirectoryRequired, got %v", err)
	}

	err = flow.Validate(&models.ExecutionParams{FindMissingSourceDir: "/source"})
	if !errors.Is(err, flows.ErrTargetDirectoryRequired) {
		t.Fatalf("expected ErrTargetDirectoryRequired, got %v", err)
	}
}

type missingContentPublisher struct {
	result      models.MissingContentResult
	diagnostics models.ScanDiagnostics
}

func (p *missingContentPublisher) PublishDuplicateResult(models.DuplicateResult) {}
func (p *missingContentPublisher) PublishMissingContentResult(result models.MissingContentResult) {
	p.result = result
}
func (p *missingContentPublisher) PublishScanDiagnostics(diagnostics models.ScanDiagnostics) {
	p.diagnostics = diagnostics
}

func TestFindMissingContentFlowRunsAgainstEmptyDirectories(t *testing.T) {
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{})
	publisher := &missingContentPublisher{}

	err := flow.Run(context.Background(), models.ExecutionParams{
		FindMissingSourceDir: t.TempDir(),
		FindMissingTargetDir: t.TempDir(),
		ResultsDir:           t.TempDir(),
		WorkerCount:          1,
		BufferSizeBytes:      1,
	}, reporting.NoOpReporter{}, publisher, flows.RunOptions{HandleDiagnostic: flows.DefaultDiagnosticHandler})

	if err != nil {
		t.Fatalf("run find-missing-content flow: %v", err)
	}
	if publisher.result.TotalSourceFiles != 0 || len(publisher.result.MissingFiles) != 0 {
		t.Fatalf("unexpected empty coverage result: %#v", publisher.result)
	}
}

func TestFindMissingContentFlowMatchesFilesThroughTheExecutionPipeline(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "covered.txt"), []byte("covered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "missing.txt"), []byte("missing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "renamed.txt"), []byte("covered"), 0o600); err != nil {
		t.Fatal(err)
	}

	flow := flows.NewFindMissingContentFlow(validation.MockValidator{})
	publisher := &missingContentPublisher{}
	err := flow.Run(context.Background(), models.ExecutionParams{
		FindMissingSourceDir: sourceDir,
		FindMissingTargetDir: targetDir,
		ResultsDir:           t.TempDir(),
		WorkerCount:          1,
		BufferSizeBytes:      1,
	}, reporting.NoOpReporter{}, publisher, flows.RunOptions{HandleDiagnostic: flows.DefaultDiagnosticHandler})
	if err != nil {
		t.Fatalf("run find-missing-content flow: %v", err)
	}
	if publisher.result.TotalSourceFiles != 2 || len(publisher.result.MissingFiles) != 1 {
		t.Fatalf("unexpected coverage result: %#v", publisher.result)
	}
	if publisher.result.MissingFiles[0].FullPath != filepath.Join(sourceDir, "missing.txt") {
		t.Fatalf("unexpected missing file: %#v", publisher.result.MissingFiles[0])
	}
}

func TestFindMissingContentFlowStopsWithoutPublishingOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{})
	publisher := &missingContentPublisher{}

	err := flow.Run(ctx, models.ExecutionParams{
		FindMissingSourceDir: t.TempDir(),
		FindMissingTargetDir: t.TempDir(),
		ResultsDir:           t.TempDir(),
		WorkerCount:          1,
		BufferSizeBytes:      1,
	}, reporting.NoOpReporter{}, publisher, flows.RunOptions{HandleDiagnostic: flows.DefaultDiagnosticHandler})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if publisher.result.TotalSourceFiles != 0 || publisher.result.MissingFiles != nil {
		t.Fatalf("expected no published coverage result, got %#v", publisher.result)
	}
}

func TestFindMissingContentFlowValidatesOnlySourceAndTarget(t *testing.T) {
	var paths []string
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{
		ReadableDirFunc: func(path string) error {
			paths = append(paths, path)
			return nil
		},
	})
	args := models.ExecutionParams{
		Directories:          []string{"/stale-duplicate-directory"},
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
	}

	err := flow.Validate(&args)

	if err != nil {
		t.Fatalf("validate find-missing-content flow: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/source" || paths[1] != "/target" {
		t.Fatalf("validated paths = %v, want [/source /target]", paths)
	}
}

func TestFindMissingContentFlowWrapsSourceReadError(t *testing.T) {
	expected := validation.ErrNoReadAccess
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{
		ReadableDirFunc: func(path string) error {
			if path == "/source" {
				return expected
			}
			return nil
		},
	})
	args := models.ExecutionParams{
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
	}

	err := flow.Validate(&args)

	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
	if !strings.HasPrefix(err.Error(), `FindMissingSourceDir ("/source")`) {
		t.Fatalf("unexpected source error: %v", err)
	}
}
