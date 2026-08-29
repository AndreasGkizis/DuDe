package flows

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"errors"
	"fmt"
)

var ErrNoDirectories = errors.New("at least one directory must be provided")

// ReadableDirectoryValidator checks that a flow input directory can be read.
type ReadableDirectoryValidator interface {
	ReadableDir(path string) error
}

// DuplicateFlow is the duplicate-finding execution flow.
type DuplicateFlow struct {
	paths ReadableDirectoryValidator
	run   func(reporter reporting.Reporter) error
}

// NewDuplicateFlow creates the duplicate flow around its current validation and
// processing functions. The injected runner is a temporary adapter until the
// duplicate orchestration moves into this package and returns a typed result.
func NewDuplicateFlow(
	paths ReadableDirectoryValidator,
	run func(reporter reporting.Reporter) error,
) DuplicateFlow {
	return DuplicateFlow{
		paths: paths,
		run:   run,
	}
}

func (DuplicateFlow) Mode() models.ExecutionMode {
	return models.ExecutionModeDuplicates
}

func (flow DuplicateFlow) Validate(args *models.ExecutionParams) error {
	if len(args.Directories) == 0 {
		return fmt.Errorf("Directories: %w", ErrNoDirectories)
	}

	for i, dir := range args.Directories {
		if err := flow.paths.ReadableDir(dir); err != nil {
			return fmt.Errorf("Directories[%d] (%q): %w", i, dir, err)
		}
	}

	return nil
}

func (flow DuplicateFlow) Run(reporter reporting.Reporter) error {
	// TODO(step 5): Replace the injected legacy runner with orchestration owned
	// here, using the processing packages established in Step 4.
	return flow.run(reporter)
}
