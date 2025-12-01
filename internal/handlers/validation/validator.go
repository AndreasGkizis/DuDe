package validation

import (
	"DuDe/internal/common/fs"
)

// IValidator defines the contract for all validation processes.
type IValidator interface {
	Resolve(value, fallback string) string
	ReadableDir(path string) error
	WritableDir(path string) error
}

type Validator struct {
	FS fs.FS
}

func (v Validator) Resolve(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (v Validator) ReadableDir(path string) error {
	if !v.FS.Exists(path) {
		return ErrPathNotExists
	}
	if !v.FS.IsDir(path) {
		return ErrPathNotDirectory
	}
	if !v.FS.CanRead(path) {
		return ErrNoReadAccess
	}
	return nil
}

func (v Validator) WritableDir(path string) error {
	if v.FS.Exists(path) {
		if !v.FS.IsDir(path) {
			return ErrPathNotDirectory
		}
		if !v.FS.CanWrite(path) {
			return ErrNoWriteAccess
		}
		return nil
	}

	parent := v.FS.Parent(path)
	if !v.FS.CanWrite(parent) {
		return ErrNoWriteAccess
	}

	return nil
}
