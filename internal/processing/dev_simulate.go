package processing

// DEV ONLY - delete this file before shipping

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

	// Done
	reporter.LogProgress(ctx, "Done", 100)
	reporter.FinishExecution(ctx)
}
