//go:build debug_progress

package processing

import "DuDe/internal/reporting"

func runSelectedExecution(app *FrontendApp, reporter reporting.Reporter) error {
	defer func() {
		if app.cancelFunc != nil {
			app.cancelFunc()
			app.cancelFunc = nil
		}
	}()

	app.lastResults = nil
	simulateExecution(app.execCtx, reporter)
	return nil
}
