package unit_test

import (
	val "DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	"errors"
	"runtime"
	"testing"
)

func setupResolver(_ *testing.T, mockV val.MockValidator) val.Resolver {
	return val.Resolver{V: mockV}
}

func resolveAndValidateArgs(r val.Resolver, args *models.ExecutionParams, exeDir string) error {
	if err := r.ValidateModeArgs(args); err != nil {
		return err
	}
	return r.ResolveSharedArgs(args, exeDir)
}

// --- Mode validation and shared resolution tests ---

func TestValidateModeArgs_RequiresExplicitMode(t *testing.T) {
	r := setupResolver(t, val.MockValidator{})

	err := r.ValidateModeArgs(&models.ExecutionParams{})

	if !errors.Is(err, val.ErrExecutionModeRequired) {
		t.Fatalf("expected ErrExecutionModeRequired, got %v", err)
	}
}

func TestValidateModeArgs_RejectsUnknownMode(t *testing.T) {
	r := setupResolver(t, val.MockValidator{})
	args := &models.ExecutionParams{Mode: models.ExecutionMode(99)}

	err := r.ValidateModeArgs(args)

	if !errors.Is(err, val.ErrUnknownExecutionMode) {
		t.Fatalf("expected ErrUnknownExecutionMode, got %v", err)
	}
}

func TestValidateModeArgs_FindMissingContentRequiresSource(t *testing.T) {
	r := setupResolver(t, val.MockValidator{})
	args := &models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingTargetDir: "/target",
	}

	err := r.ValidateModeArgs(args)

	if !errors.Is(err, val.ErrSourceDirectoryRequired) {
		t.Fatalf("expected ErrSourceDirectoryRequired, got %v", err)
	}
}

func TestValidateModeArgs_FindMissingContentRequiresTarget(t *testing.T) {
	r := setupResolver(t, val.MockValidator{})
	args := &models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingSourceDir: "/source",
	}

	err := r.ValidateModeArgs(args)

	if !errors.Is(err, val.ErrTargetDirectoryRequired) {
		t.Fatalf("expected ErrTargetDirectoryRequired, got %v", err)
	}
}

func TestValidateModeArgs_FindMissingContentValidatesOnlySourceAndTarget(t *testing.T) {
	var readablePaths []string
	r := setupResolver(t, val.MockValidator{
		ReadableDirFunc: func(path string) error {
			readablePaths = append(readablePaths, path)
			return nil
		},
	})
	args := &models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		Directories:          []string{"/stale/duplicate-directory"},
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
	}

	err := r.ValidateModeArgs(args)

	if err != nil {
		t.Fatalf("expected valid find-missing-content arguments, got %v", err)
	}
	want := []string{"/source", "/target"}
	if len(readablePaths) != len(want) {
		t.Fatalf("readable path count = %d, want %d (%v)", len(readablePaths), len(want), readablePaths)
	}
	for i := range want {
		if readablePaths[i] != want[i] {
			t.Fatalf("readablePaths[%d] = %q, want %q", i, readablePaths[i], want[i])
		}
	}
}

func TestValidateModeArgs_FindMissingContentWrapsSourceError(t *testing.T) {
	r := setupResolver(t, val.MockValidator{
		ReadableDirFunc: func(path string) error {
			if path == "/source" {
				return val.ErrNoReadAccess
			}
			return nil
		},
	})
	args := &models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
	}

	err := r.ValidateModeArgs(args)

	if !errors.Is(err, val.ErrNoReadAccess) {
		t.Fatalf("expected wrapped source read error, got %v", err)
	}
	if want := `FindMissingSourceDir ("/source")`; err == nil || len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Fatalf("source error = %v, want prefix %q", err, want)
	}
}

func TestValidateModeArgs_FindMissingContentWrapsTargetError(t *testing.T) {
	r := setupResolver(t, val.MockValidator{
		ReadableDirFunc: func(path string) error {
			if path == "/target" {
				return val.ErrPathNotExists
			}
			return nil
		},
	})
	args := &models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
	}

	err := r.ValidateModeArgs(args)

	if !errors.Is(err, val.ErrPathNotExists) {
		t.Fatalf("expected wrapped target path error, got %v", err)
	}
	if want := `FindMissingTargetDir ("/target")`; err == nil || len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Fatalf("target error = %v, want prefix %q", err, want)
	}
}

func TestResolveSharedArgs_DoesNotRequireModeOrDirectories(t *testing.T) {
	r := setupResolver(t, val.MockValidator{
		ReadableDirFunc: func(path string) error {
			t.Fatalf("shared resolution unexpectedly validated readable path %q", path)
			return nil
		},
		WritableDirFunc: func(path string) error { return nil },
	})
	args := &models.ExecutionParams{}

	err := r.ResolveSharedArgs(args, "/fallback")

	if err != nil {
		t.Fatalf("expected shared resolution to succeed, got %v", err)
	}
	if args.CacheDir != "/fallback" {
		t.Fatalf("CacheDir = %q, want fallback", args.CacheDir)
	}
	if args.ResultsDir != "/fallback" {
		t.Fatalf("ResultsDir = %q, want fallback", args.ResultsDir)
	}
	if args.WorkerCount != runtime.GOMAXPROCS(0) {
		t.Fatalf("WorkerCount = %d, want %d", args.WorkerCount, runtime.GOMAXPROCS(0))
	}
	if args.BufferSizeBytes != 1024 {
		t.Fatalf("BufferSizeBytes = %d, want 1024", args.BufferSizeBytes)
	}
}

func TestValidateModeAndResolveShared_Success(t *testing.T) {
	mockV := val.MockValidator{
		// All paths are fine
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	exeDir := "/exe/bin"
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/app/src", "/app/target"},
		CacheDir:    "", // Should fall back to exeDir
		ResultsDir:  "custom/results",
	}

	err := resolveAndValidateArgs(r, args, exeDir)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	// Check if defaults were applied correctly
	if args.CacheDir != exeDir {
		t.Errorf("CacheDir not resolved correctly. Got %s, Expected %s", args.CacheDir, exeDir)
	}
}

func TestValidateModeAndResolveShared_Fails_SourceDir(t *testing.T) {
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
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/bad/source"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check for correct error wrapping
	if !errors.Is(err, expectedErr) {
		t.Errorf("Error missing wrapped error. Expected %v, got %v", expectedErr, err)
	}
	if expectedPrefix := "Directories["; !errors.Is(err, expectedErr) && err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Error message not wrapped correctly. Got %s", err.Error())
	}
}

func TestValidateModeAndResolveShared_AllowsUnavailableOptionalCache(t *testing.T) {
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error {
			if p == "/bad/cache" {
				return val.ErrPathNotDirectory
			}
			return nil
		},
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"ok/src"},
		UseCache:    true,
		CacheDir:    "/bad/cache",
		ResultsDir:  "ok/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err != nil {
		t.Fatalf("expected unavailable optional cache not to block execution, got %v", err)
	}
}

func TestValidateModeAndResolveShared_Fails_SecondDir(t *testing.T) {
	expectedErr := val.ErrNoReadAccess
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error {
			if p == "/bad/target" {
				return expectedErr
			}
			return nil
		},
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/ok/source", "/bad/target"},
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err == nil || !errors.Is(err, expectedErr) {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestValidateModeAndResolveShared_EmptyDirectories(t *testing.T) {
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{}, // empty — must fail
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err == nil {
		t.Fatal("Expected error for empty Directories, got nil")
	}
	if !errors.Is(err, val.ErrNoDirectories) {
		t.Errorf("Expected ErrNoDirectories, got: %v", err)
	}
}

func TestValidateModeAndResolveShared_ThreeDirs(t *testing.T) {
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/dir/one", "/dir/two", "/dir/three"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err != nil {
		t.Fatalf("Expected nil error for three valid dirs, got %v", err)
	}
}

func TestValidateModeAndResolveShared_ThreeDirs_OneInvalid(t *testing.T) {
	expectedErr := val.ErrNoReadAccess
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error {
			if p == "/dir/bad" {
				return expectedErr
			}
			return nil
		},
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/dir/one", "/dir/bad", "/dir/three"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err == nil {
		t.Fatal("Expected error for invalid directory, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected ErrNoReadAccess, got: %v", err)
	}
}

func TestValidateModeAndResolveShared_Fails_ResultsDir(t *testing.T) {
	expectedErr := errors.New("permission denied")
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error {
			if p == "/bad/results" {
				return expectedErr
			}
			return nil
		},
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Mode:        models.ExecutionModeDuplicates,
		Directories: []string{"/ok/source"},
		ResultsDir:  "/bad/results",
	}

	err := resolveAndValidateArgs(r, args, "/exe")
	if err == nil || !errors.Is(err, expectedErr) {
		t.Fatalf("Expected ResultsDir error, got %v", err)
	}
}
func TestResolveWorkers(t *testing.T) {
	maxCPUs := runtime.GOMAXPROCS(0)

	mockV := val.MockValidator{
		// All paths are fine
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)

	testCases := []struct {
		name     string
		params   models.ExecutionParams
		expected models.ExecutionParams
	}{
		{
			name:     "Zero value defaults to Max",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, WorkerCount: 0},
			expected: models.ExecutionParams{WorkerCount: maxCPUs},
		},
		{
			name:     "Negative value defaults to Max",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, WorkerCount: -5},
			expected: models.ExecutionParams{WorkerCount: maxCPUs},
		},
		{
			name:     "Valid value within range",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, WorkerCount: 2},
			expected: models.ExecutionParams{WorkerCount: 2},
		},
		{
			name:     "Value exceeding Max is capped",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, WorkerCount: maxCPUs + 1},
			expected: models.ExecutionParams{WorkerCount: maxCPUs},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// We pass the address of the field from the struct instance
			err := resolveAndValidateArgs(r, &tt.params, "")
			if err != nil {
				t.Errorf("%s: Some error %d", tt.name, err)
			}
			if tt.params.WorkerCount != tt.expected.WorkerCount {
				t.Errorf("Expected %d but got %d", tt.expected.WorkerCount, tt.params.WorkerCount)
			}
		})
	}
}

func TestResolveBufferSize(t *testing.T) {
	const defaultBuf = 1024
	const maxValue = 1048576

	mockV := val.MockValidator{
		// All paths are fine
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	testCases := []struct {
		name     string
		params   models.ExecutionParams
		expected models.ExecutionParams
	}{
		{
			name:     "Zero value defaults",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, BufferSizeBytes: 0},
			expected: models.ExecutionParams{BufferSizeBytes: defaultBuf},
		},
		{
			name:     "Negative value defaults",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, BufferSizeBytes: -1},
			expected: models.ExecutionParams{BufferSizeBytes: defaultBuf},
		},
		{
			name:     "Specific valid buffer",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, BufferSizeBytes: 2048},
			expected: models.ExecutionParams{BufferSizeBytes: 2048},
		},
		{
			name:     "Specific valid buffer, over max",
			params:   models.ExecutionParams{Mode: models.ExecutionModeDuplicates, Directories: []string{"/placeholder"}, BufferSizeBytes: 100000000000},
			expected: models.ExecutionParams{BufferSizeBytes: maxValue},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// We pass the address of the field from the struct instance
			err := resolveAndValidateArgs(r, &tt.params, "")
			if err != nil {
				t.Errorf("%s: Some error %d", tt.name, err)
			}
			if tt.params.BufferSizeBytes != tt.expected.BufferSizeBytes {
				t.Errorf("Expected %d but got %d", tt.expected.BufferSizeBytes, tt.params.BufferSizeBytes)
			}
		})
	}
}
