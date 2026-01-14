package reporting

import "context"

// Reporter defines the methods required to update the frontend.
// The processing package can use this interface without importing processing.
type Reporter interface {
	LogProgress(ctx context.Context, title string, percent float64)
	LogDetailedStatus(ctx context.Context, message string)
	FinishExecution(ctx context.Context)
}

type ProgressUpdate struct {
	Title   string  `json:"title"`
	Percent float64 `json:"percent"`
}
