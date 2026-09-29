package flows

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"fmt"
)

var (
	ErrSourceDirectoryRequired = errors.New("source directory must be provided")
	ErrTargetDirectoryRequired = errors.New("target directory must be provided")
)

// FindMissingContentFlow owns Find Missing Content validation and execution.
// It is intentionally not registered until its backend produces a typed result.
type FindMissingContentFlow struct {
	paths ReadableDirectoryValidator
}

func NewFindMissingContentFlow(paths ReadableDirectoryValidator) FindMissingContentFlow {
	return FindMissingContentFlow{paths: paths}
}

func (FindMissingContentFlow) Mode() models.ExecutionMode {
	return models.ExecutionModeFindMissingContent
}

func (flow FindMissingContentFlow) Validate(args *models.ExecutionParams) error {
	if args.FindMissingSourceDir == "" {
		return fmt.Errorf("FindMissingSourceDir: %w", ErrSourceDirectoryRequired)
	}
	if args.FindMissingTargetDir == "" {
		return fmt.Errorf("FindMissingTargetDir: %w", ErrTargetDirectoryRequired)
	}

	if err := flow.paths.ReadableDir(args.FindMissingSourceDir); err != nil {
		return fmt.Errorf("FindMissingSourceDir (%q): %w", args.FindMissingSourceDir, err)
	}
	if err := flow.paths.ReadableDir(args.FindMissingTargetDir); err != nil {
		return fmt.Errorf("FindMissingTargetDir (%q): %w", args.FindMissingTargetDir, err)
	}

	return nil
}

func (FindMissingContentFlow) Run(context.Context, models.ExecutionParams, reporting.Reporter, ResultPublisher) error {
	return ErrExecutionModeUnavailable
}
