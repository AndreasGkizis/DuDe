//go:build debug_progress

package flows

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrNoDirectories = errors.New("at least one directory must be provided")

type ReadableDirectoryValidator interface {
	ReadableDir(path string) error
}

// DuplicateFlow is duplicated here intentionally: this development-only build
// replaces the production duplicate flow as a complete unit.
type DuplicateFlow struct {
	paths ReadableDirectoryValidator
}

func NewDuplicateFlow(paths ReadableDirectoryValidator) DuplicateFlow {
	return DuplicateFlow{paths: paths}
}

func (DuplicateFlow) Mode() models.ExecutionMode {
	return models.ExecutionModeDuplicates
}

func (flow DuplicateFlow) Validate(args *models.ExecutionParams) error {
	if len(args.Directories) == 0 {
		return fmt.Errorf("Directories: %w", ErrNoDirectories)
	}
	for i, dir := range args.Directories {
		if err := flow.paths.ReadableDir(dir); err != nil {
			return fmt.Errorf("Directories[%d] (%q): %w", i, dir, err)
		}
	}
	return nil
}

// Run is a development-only progress simulator.
func (flow DuplicateFlow) Run(ctx context.Context, _ models.ExecutionParams, reporter reporting.Reporter, publisher ResultPublisher) error {
	simulateExecution(ctx, reporter)
	if err := ctx.Err(); err != nil {
		return err
	}
	publisher.PublishDuplicateResult(models.DuplicateResult{})
	return nil
}

func simulateExecution(ctx context.Context, reporter reporting.Reporter) {
	const total = int64(120)

	reporter.LogProgress(ctx, "Reading", 0)
	for i := int64(1); i <= total; i++ {
		if !waitForProgress(ctx, 30*time.Millisecond) {
			return
		}
		reporter.LogFilesCount(ctx, i, 0)
	}
	reporter.LogProgress(ctx, "Reading", 100)

	reporter.LogProgress(ctx, "Filtering", 0)
	reporter.LogFilesCount(ctx, 0, total)
	if !waitForProgress(ctx, 500*time.Millisecond) {
		return
	}
	reporter.LogFilesCount(ctx, total, total)
	reporter.LogProgress(ctx, "Filtering", 100)

	for _, phase := range []struct {
		name  string
		delay time.Duration
	}{
		{"Hashing", 40 * time.Millisecond}, {"Caching", 10 * time.Millisecond},
		{"Finding", 15 * time.Millisecond}, {"Comparing", 15 * time.Millisecond},
	} {
		reporter.LogProgress(ctx, phase.name, 0)
		for i := int64(1); i <= total; i++ {
			if !waitForProgress(ctx, phase.delay) {
				return
			}
			reporter.LogFilesCount(ctx, i, total)
			reporter.LogProgress(ctx, phase.name, float64(i)/float64(total)*100)
		}
	}

	reporter.LogProgress(ctx, "Collecting", 0)
	reporter.LogFilesCount(ctx, 0, total)
	if !waitForProgress(ctx, 500*time.Millisecond) {
		return
	}
	reporter.LogFilesCount(ctx, total, total)
	reporter.LogProgress(ctx, "Collecting", 100)
	reporter.LogProgress(ctx, "Done", 100)
	reporter.FinishExecution(ctx)
}

func waitForProgress(ctx context.Context, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}
