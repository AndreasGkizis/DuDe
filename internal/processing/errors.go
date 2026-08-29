package processing

import (
	"DuDe/internal/processing/flows"
	"errors"
)

var (
	ErrNoFilesFound             = errors.New("execution skipped: no files found in the specified directories")
	ErrExecutionModeUnavailable = flows.ErrExecutionModeUnavailable
)
