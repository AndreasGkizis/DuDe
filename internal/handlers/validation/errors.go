package validation

import "errors"

var (
	ErrPathNotDirectory = errors.New("path is not a directory")
	ErrPathNotExists    = errors.New("path does not exist")
	ErrNoReadAccess     = errors.New("no read access")
	ErrNoWriteAccess    = errors.New("no write access")
)
