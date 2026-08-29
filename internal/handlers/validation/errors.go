package validation

import "errors"

var (
	ErrPathNotDirectory        = errors.New("path is not a directory")
	ErrPathNotExists           = errors.New("path does not exist")
	ErrNoReadAccess            = errors.New("no read access")
	ErrNoWriteAccess           = errors.New("no write access")
	ErrNoDirectories           = errors.New("at least one directory must be provided")
	ErrExecutionModeRequired   = errors.New("execution mode must be provided")
	ErrUnknownExecutionMode    = errors.New("unknown execution mode")
	ErrSourceDirectoryRequired = errors.New("source directory must be provided")
	ErrTargetDirectoryRequired = errors.New("target directory must be provided")
)
