package processing

import "errors"

var (
	ErrNoFilesFound             = errors.New("execution skipped: no files found in the specified directories")
	ErrExecutionModeUnavailable = errors.New("execution mode is not connected yet")
)
