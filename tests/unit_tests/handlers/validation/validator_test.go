// validation/validator_test.go
package unit_test

import (
	fs "DuDe/internal/common/fs"
	val "DuDe/internal/handlers/validation" // Assuming your validation package is named 'validation'
	"errors"
	"testing"
)

// Helper to set up a Validator with the mock
func setupValidator(_ *testing.T, mockFS fs.MockFS) val.Validator {
	// Note: We use the actual package name for injection
	return val.Validator{FS: mockFS}
}

// --- ReadableDir Tests ---

func TestReadableDir_Success(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:  func(p string) bool { return true },
		IsDirFunc:   func(p string) bool { return true },
		CanReadFunc: func(p string) bool { return true },
	}
	v := setupValidator(t, mockFS)

	err := v.ReadableDir("/existing/readable/dir")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestReadableDir_Fails_PathNotExists(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc: func(p string) bool { return false },
	}
	v := setupValidator(t, mockFS)

	err := v.ReadableDir("/non/existent/path")
	if !errors.Is(err, val.ErrPathNotExists) {
		t.Errorf("Expected ErrPathNotExists, got %v", err)
	}
}

func TestReadableDir_Fails_PathNotDirectory(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc: func(p string) bool { return true },
		IsDirFunc:  func(p string) bool { return false },
	}
	v := setupValidator(t, mockFS)

	err := v.ReadableDir("/existing/file")
	if !errors.Is(err, val.ErrPathNotDirectory) {
		t.Errorf("Expected ErrPathNotDirectory, got %v", err)
	}
}

func TestReadableDir_Fails_NoReadAccess(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:  func(p string) bool { return true },
		IsDirFunc:   func(p string) bool { return true },
		CanReadFunc: func(p string) bool { return false },
	}
	v := setupValidator(t, mockFS)

	err := v.ReadableDir("/existing/unreadable/dir")
	if !errors.Is(err, val.ErrNoReadAccess) {
		t.Errorf("Expected ErrNoReadAccess, got %v", err)
	}
}

// --- WritableDir Tests ---

func TestWritableDir_Success_ExistingDir(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:   func(p string) bool { return true },
		IsDirFunc:    func(p string) bool { return true },
		CanWriteFunc: func(p string) bool { return true },
	}
	v := setupValidator(t, mockFS)

	err := v.WritableDir("/existing/writable/dir")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestWritableDir_Success_NewDir_WritableParent(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:   func(p string) bool { return p == "/parent" }, // Only parent exists
		IsDirFunc:    func(p string) bool { return true },
		CanWriteFunc: func(p string) bool { return p == "/parent" }, // Only parent is writable
		ParentFunc:   func(p string) string { return "/parent" },
	}
	v := setupValidator(t, mockFS)

	err := v.WritableDir("/parent/new-dir")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestWritableDir_Fails_ExistingPathNotDirectory(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc: func(p string) bool { return true },
		IsDirFunc:  func(p string) bool { return false },
	}
	v := setupValidator(t, mockFS)

	err := v.WritableDir("/existing/file")
	if !errors.Is(err, val.ErrPathNotDirectory) {
		t.Errorf("Expected ErrPathNotDirectory, got %v", err)
	}
}

func TestWritableDir_Fails_ExistingDirNoWriteAccess(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:   func(p string) bool { return true },
		IsDirFunc:    func(p string) bool { return true },
		CanWriteFunc: func(p string) bool { return false },
	}
	v := setupValidator(t, mockFS)

	err := v.WritableDir("/existing/read-only/dir")
	if !errors.Is(err, val.ErrNoWriteAccess) {
		t.Errorf("Expected ErrNoWriteAccess, got %v", err)
	}
}

func TestWritableDir_Fails_NewDir_UnwritableParent(t *testing.T) {
	mockFS := fs.MockFS{
		ExistsFunc:   func(p string) bool { return p == "/parent" },
		IsDirFunc:    func(p string) bool { return true },
		CanWriteFunc: func(p string) bool { return false }, // Parent is NOT writable
		ParentFunc:   func(p string) string { return "/parent" },
	}
	v := setupValidator(t, mockFS)

	err := v.WritableDir("/parent/new-dir")
	if !errors.Is(err, val.ErrNoWriteAccess) {
		t.Errorf("Expected ErrNoWriteAccess, got %v", err)
	}
}
