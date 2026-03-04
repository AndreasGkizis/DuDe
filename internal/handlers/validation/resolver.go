package validation

import (
	"DuDe/internal/models"
	"fmt"
	"runtime"
)

type Resolver struct {
	V IValidator
}

func (r Resolver) ResolveAndValidateArgs(args *models.ExecutionParams, exeDir string) error {
	args.CacheDir = resolveDir(args.CacheDir, exeDir)
	args.ResultsDir = resolveDir(args.ResultsDir, exeDir)

	// At least one directory must be provided
	if len(args.Directories) == 0 {
		return fmt.Errorf("Directories: %w", ErrNoDirectories)
	}

	// Validate every supplied directory is readable
	for i, dir := range args.Directories {
		if err := r.V.ReadableDir(dir); err != nil {
			return fmt.Errorf("Directories[%d] (%q): %w", i, dir, err)
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

	// resolve or validate the cpus
	args.CPUs = resolveWorkers(&args.CPUs)

	args.BufSize = resolveBufferSize(&args.BufSize)

	return nil
}

func resolveDir(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func resolveWorkers(value *int) int {
	max := runtime.GOMAXPROCS(0)

	if value == nil || *value <= 0 {
		return max // Default to max performance if not specified
	}

	if *value > max {
		return max
	}

	return *value

}

func resolveBufferSize(value *int) int {
	const defaultValue = 1024
	const maxValue = 1048576

	if value == nil || *value <= 0 {
		return defaultValue // Default to max performance if not specified
	}
	if *value > maxValue {
		return maxValue
	}

	return *value

}
