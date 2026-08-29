package validation

import (
	"DuDe/internal/models"
	"fmt"
	"runtime"
)

type Resolver struct {
	V IValidator
}

// ResolveSharedArgs resolves and validates options shared by every execution mode.
func (r Resolver) ResolveSharedArgs(args *models.ExecutionParams, exeDir string) error {
	args.CacheDir = resolveDir(args.CacheDir, exeDir)
	args.ResultsDir = resolveDir(args.ResultsDir, exeDir)

	// ResultsDir (writable, parent fallback)
	if err := r.V.WritableDir(args.ResultsDir); err != nil {
		return fmt.Errorf("ResultsDir: %w", err)
	}

	// resolve or validate the cpus
	args.WorkerCount = resolveWorkers(&args.WorkerCount)

	args.BufferSizeBytes = resolveBufferSize(&args.BufferSizeBytes)

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
