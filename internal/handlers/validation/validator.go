package validation

import (
	"DuDe/internal/common/fs"
)

// PathValidator checks filesystem access needed by execution arguments.
type PathValidator interface {
	ReadableDir(path string) error
	WritableDir(path string) error
}

type FilesystemValidator struct {
	FS fs.FS
}

func (v FilesystemValidator) ReadableDir(path string) error {
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

func (v FilesystemValidator) WritableDir(path string) error {
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
