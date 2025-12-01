package reporting

import "context"

// Reporter defines the methods required to update the frontend.
// The processing package can use this interface without importing processing.
type Reporter interface {
	LogProgress(ctx context.Context, title string, percent int)
	LogDetailedStatus(ctx context.Context, message string)
}

type ProgressUpdate struct {
	Title   string `json:"title"`
	Percent int    `json:"percent"`
}
