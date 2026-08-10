//go:build !debug_progress

package processing

import "DuDe/internal/reporting"

func runSelectedExecution(app *FrontendApp, reporter reporting.Reporter) error {
	return startExecution(app, reporter)
}
