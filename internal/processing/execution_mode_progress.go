//go:build debug_progress

package processing

import "DuDe/internal/reporting"

func runSelectedExecution(app *FrontendApp, reporter reporting.Reporter) error {
	app.setLastResults(nil)
	simulateExecution(app.executionContext(), reporter)
	return nil
}
