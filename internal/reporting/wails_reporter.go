package reporting

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WailsReporter struct{}

// LogDetailedStatus sends continuous log messages to the detailed status box.
func (a *WailsReporter) LogDetailedStatus(ctx context.Context, message string) {
	// Use a new event name specifically for detailed logging
	runtime.EventsEmit(ctx, "detailedLog", message)
}

// LogProgress sends progress percentage and title to the frontend.
func (a *WailsReporter) LogProgress(ctx context.Context, title string, percent float64) {
	update := ProgressUpdate{
		Title:   title,
		Percent: percent,
	}
	// Wails automatically marshals the struct to JSON
	runtime.EventsEmit(ctx, "progressUpdate", update)
}

// FinishExecution signals the endof execution to the frontend.
func (a *WailsReporter) FinishExecution(ctx context.Context) {
	runtime.EventsEmit(ctx, "executionFinished")
}
