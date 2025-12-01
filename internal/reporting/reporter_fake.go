package reporting

import (
	"context"
	"testing"
)

// NoOpReporter implements the reporting.Reporter interface for E2E testing.
// It accepts a *testing.T to optionally log activities for debugging,
// but crucially avoids any calls to the Wails runtime.
type NoOpReporter struct {
	*testing.T
}

// LogDetailedStatus satisfies the interface contract but executes no Wails code.
func (l NoOpReporter) LogDetailedStatus(ctx context.Context, message string) {
	// Optional: Log to test output for debugging concurrency/flow
	// l.T.Logf("E2E Detailed Log: %s", message)
}

// LogProgress satisfies the interface contract but executes no Wails code.
func (l NoOpReporter) LogProgress(ctx context.Context, title string, percent int) {
	// l.T.Logf("E2E Progress: %s - %d%%", title, percent)
}
