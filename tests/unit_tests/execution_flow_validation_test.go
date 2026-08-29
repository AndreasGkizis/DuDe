package unit_tests

import (
	"DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	"DuDe/internal/processing/flows"
	"DuDe/internal/reporting"
	"errors"
	"strings"
	"testing"
)

func noOpFlowRun(reporting.Reporter) error { return nil }

func TestDuplicateFlowRequiresDirectories(t *testing.T) {
	flow := flows.NewDuplicateFlow(validation.MockValidator{}, noOpFlowRun)

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
	}, noOpFlowRun)
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
	}, noOpFlowRun)
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
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{}, noOpFlowRun)

	err := flow.Validate(&models.ExecutionParams{})
	if !errors.Is(err, flows.ErrSourceDirectoryRequired) {
		t.Fatalf("expected ErrSourceDirectoryRequired, got %v", err)
	}

	err = flow.Validate(&models.ExecutionParams{FindMissingSourceDir: "/source"})
	if !errors.Is(err, flows.ErrTargetDirectoryRequired) {
		t.Fatalf("expected ErrTargetDirectoryRequired, got %v", err)
	}
}

func TestFindMissingContentFlowValidatesOnlySourceAndTarget(t *testing.T) {
	var paths []string
	flow := flows.NewFindMissingContentFlow(validation.MockValidator{
		ReadableDirFunc: func(path string) error {
			paths = append(paths, path)
			return nil
		},
	}, noOpFlowRun)
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
	}, noOpFlowRun)
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
