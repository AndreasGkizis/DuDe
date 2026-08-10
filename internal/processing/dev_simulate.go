//go:build debug_progress

package processing

import (
	"DuDe/internal/reporting"
	"context"
	"time"
)

func simulateExecution(ctx context.Context, reporter reporting.Reporter) {

	const total = int64(120)

	// Phase 1: Reading (total unknown)
	reporter.LogProgress(ctx, "Reading", 0)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(30 * time.Millisecond)
		reporter.LogFilesCount(ctx, i, 0)
	}
	reporter.LogProgress(ctx, "Reading", 100)

	reporter.LogProgress(ctx, "Filtering", 0)
	time.Sleep(500 * time.Millisecond)
	reporter.LogProgress(ctx, "Filtering", 100)

	// Phase 2: Hashing
	reporter.LogProgress(ctx, "Hashing", 0)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(40 * time.Millisecond)
		reporter.LogFilesCount(ctx, i, total)
		reporter.LogProgress(ctx, "Hashing", float64(i)/float64(total)*100)
	}

	reporter.LogProgress(ctx, "Caching", 0)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(10 * time.Millisecond)
		reporter.LogFilesCount(ctx, i, total)
		reporter.LogProgress(ctx, "Caching", float64(i)/float64(total)*100)
	}

	// Phase 3: Finding
	reporter.LogProgress(ctx, "Finding", 0)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(15 * time.Millisecond)
		reporter.LogFilesCount(ctx, i, total)
		reporter.LogProgress(ctx, "Finding", float64(i)/float64(total)*100)
	}

	reporter.LogProgress(ctx, "Collecting", 0)
	time.Sleep(500 * time.Millisecond)
	reporter.LogProgress(ctx, "Collecting", 100)

	reporter.LogProgress(ctx, "Comparing", 0)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(15 * time.Millisecond)
		reporter.LogFilesCount(ctx, i, total)
		reporter.LogProgress(ctx, "Comparing", float64(i)/float64(total)*100)
	}

	// Done
	reporter.LogProgress(ctx, "Done", 100)
	reporter.FinishExecution(ctx)
}
