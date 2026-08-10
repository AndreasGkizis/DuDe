//go:build debug_progress

package processing

import "DuDe/internal/reporting"

func runSelectedExecution(app *FrontendApp, reporter reporting.Reporter) error {
	app.lastResults = nil
	simulateExecution(app.execCtx, reporter)
	return nil
}
