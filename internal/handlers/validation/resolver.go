package validation

import (
	"DuDe/internal/models"
	"fmt"
)

type Resolver struct {
	V IValidator
}

func (r Resolver) ResolveAndValidateArgs(args *models.ExecutionParams, exeDir string) error {
	args.CacheDir = r.V.Resolve(args.CacheDir, exeDir)
	args.ResultsDir = r.V.Resolve(args.ResultsDir, exeDir)

	// SourceDir (must exist + read)
	if err := r.V.ReadableDir(args.SourceDir); err != nil {
		return fmt.Errorf("SourceDir: %w", err)
	}

	// TargetDir optional
	if args.TargetDir != "" {
		if err := r.V.ReadableDir(args.TargetDir); err != nil {
			return fmt.Errorf("TargetDir: %w", err)
		}
	}

	// CacheDir (writable, parent fallback)
	if err := r.V.WritableDir(args.CacheDir); err != nil {
		return fmt.Errorf("CacheDir: %w", err)
	}

	// ResultsDir (writable, parent fallback)
	if err := r.V.WritableDir(args.ResultsDir); err != nil {
		return fmt.Errorf("ResultsDir: %w", err)
	}

	return nil
}
