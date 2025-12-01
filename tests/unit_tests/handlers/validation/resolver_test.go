package unit_test

import (
	val "DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	"errors"
	"testing"
)

func setupResolver(_ *testing.T, mockV val.MockValidator) val.Resolver {
	return val.Resolver{V: mockV}
}

// --- ResolveAndValidateArgs Tests ---

func TestResolveAndValidateArgs_Success(t *testing.T) {
	mockV := val.MockValidator{
		// All paths are fine
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	exeDir := "/exe/bin"
	args := &models.ExecutionParams{
		SourceDir:  "/app/src",
		TargetDir:  "/app/target",
		CacheDir:   "", // Should fall back to exeDir
		ResultsDir: "custom/results",
	}

	err := r.ResolveAndValidateArgs(args, exeDir)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	// Check if defaults were applied correctly
	if args.CacheDir != exeDir {
		t.Errorf("CacheDir not resolved correctly. Got %s, Expected %s", args.CacheDir, exeDir)
	}
}

func TestResolveAndValidateArgs_Fails_SourceDir(t *testing.T) {
	expectedErr := val.ErrNoReadAccess
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error {
			if p == "/bad/source" {
				return expectedErr
			}
			return nil
		},
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		SourceDir:  "/bad/source",
		CacheDir:   "ok/cache",
		ResultsDir: "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check for correct error wrapping
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error missing wrapped error. Expected %v, got %v", expectedErr, err)
	}
	if expectedPrefix := "SourceDir: "; !errors.Is(err, expectedErr) && err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Error message not wrapped correctly. Got %s", err.Error())
	}
}

func TestResolveAndValidateArgs_Fails_CacheDir(t *testing.T) {
	expectedErr := val.ErrPathNotDirectory
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error {
			if p == "/bad/cache" {
				return expectedErr
			}
			return nil
		},
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		SourceDir:  "ok/src",
		CacheDir:   "/bad/cache", // Should fail here
		ResultsDir: "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check for correct error wrapping
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error missing wrapped error. Expected %v, got %v", expectedErr, err)
	}
	if expectedPrefix := "CacheDir: "; !errors.Is(err, expectedErr) && err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Error message not wrapped correctly. Got %s", err.Error())
	}
}
