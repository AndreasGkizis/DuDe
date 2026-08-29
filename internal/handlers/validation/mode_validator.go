package validation

import (
	"DuDe/internal/models"
	"fmt"
)

// ValidateModeArgs validates only the arguments owned by the selected mode.
func (r Resolver) ValidateModeArgs(args *models.ExecutionParams) error {
	switch args.Mode {
	case 0:
		return fmt.Errorf("Mode: %w", ErrExecutionModeRequired)
	case models.ExecutionModeDuplicates:
		return r.validateDuplicateArgs(args)
	case models.ExecutionModeFindMissingContent:
		return r.validateFindMissingContentArgs(args)
	default:
		return fmt.Errorf("Mode (%d): %w", args.Mode, ErrUnknownExecutionMode)
	}
}

func (r Resolver) validateDuplicateArgs(args *models.ExecutionParams) error {
	if len(args.Directories) == 0 {
		return fmt.Errorf("Directories: %w", ErrNoDirectories)
	}

	for i, dir := range args.Directories {
		if err := r.V.ReadableDir(dir); err != nil {
			return fmt.Errorf("Directories[%d] (%q): %w", i, dir, err)
		}
	}

	return nil
}

func (r Resolver) validateFindMissingContentArgs(args *models.ExecutionParams) error {
	if args.FindMissingSourceDir == "" {
		return fmt.Errorf("FindMissingSourceDir: %w", ErrSourceDirectoryRequired)
	}
	if args.FindMissingTargetDir == "" {
		return fmt.Errorf("FindMissingTargetDir: %w", ErrTargetDirectoryRequired)
	}

	if err := r.V.ReadableDir(args.FindMissingSourceDir); err != nil {
		return fmt.Errorf("FindMissingSourceDir (%q): %w", args.FindMissingSourceDir, err)
	}
	if err := r.V.ReadableDir(args.FindMissingTargetDir); err != nil {
		return fmt.Errorf("FindMissingTargetDir (%q): %w", args.FindMissingTargetDir, err)
	}

	return nil
}
