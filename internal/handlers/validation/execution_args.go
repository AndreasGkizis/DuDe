package validation

import (
	"DuDe/internal/models"
	"fmt"
	"runtime"
)

// ResolveExecutionArgs applies shared defaults and limits to args.
// Resolution changes values but does not validate filesystem access.
func ResolveExecutionArgs(args *models.ExecutionParams, fallbackDir string) {
	args.CacheDir = resolveDir(args.CacheDir, fallbackDir)
	args.ResultsDir = resolveDir(args.ResultsDir, fallbackDir)
	args.WorkerCount = resolveWorkers(args.WorkerCount)
	args.BufferSizeBytes = resolveBufferSize(args.BufferSizeBytes)
}

// ValidateSharedExecutionArgs validates fields used by every execution flow.
func ValidateSharedExecutionArgs(args *models.ExecutionParams, paths PathValidator) error {
	if err := paths.WritableDir(args.ResultsDir); err != nil {
		return fmt.Errorf("ResultsDir: %w", err)
	}
	return nil
}

func resolveDir(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func resolveWorkers(value int) int {
	max := runtime.GOMAXPROCS(0)
	if value <= 0 || value > max {
		return max
	}
	return value
}

func resolveBufferSize(value int) int {
	const defaultValue = 1024
	const maxValue = 1048576

	if value <= 0 {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
