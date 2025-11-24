package reporting

// Reporter defines the methods required to update the frontend.
// The processing package can use this interface without importing processing.
type Reporter interface {
	LogProgress(title string, percent int)
	LogDetailedStatus(message string)
}
