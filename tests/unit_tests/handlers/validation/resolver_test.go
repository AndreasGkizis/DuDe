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
		Directories: []string{"/app/src", "/app/target"},
		CacheDir:    "", // Should fall back to exeDir
		ResultsDir:  "custom/results",
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
		Directories: []string{"/bad/source"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
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
		Directories: []string{"ok/src"},
		CacheDir:    "/bad/cache", // Should fail here
		ResultsDir:  "ok/results",
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

func TestResolveAndValidateArgs_Fails_SecondDir(t *testing.T) {
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
		Directories: []string{"/ok/source", "/bad/target"},
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err == nil || !errors.Is(err, expectedErr) {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestResolveAndValidateArgs_EmptyDirectories(t *testing.T) {
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Directories: []string{}, // empty — must fail
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err == nil {
		t.Fatal("Expected error for empty Directories, got nil")
	}
	if !errors.Is(err, val.ErrNoDirectories) {
		t.Errorf("Expected ErrNoDirectories, got: %v", err)
	}
}

func TestResolveAndValidateArgs_ThreeDirs(t *testing.T) {
	mockV := val.MockValidator{
		ReadableDirFunc: func(p string) error { return nil },
		WritableDirFunc: func(p string) error { return nil },
	}
	r := setupResolver(t, mockV)
	args := &models.ExecutionParams{
		Directories: []string{"/dir/one", "/dir/two", "/dir/three"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err != nil {
		t.Fatalf("Expected nil error for three valid dirs, got %v", err)
	}
}

func TestResolveAndValidateArgs_ThreeDirs_OneInvalid(t *testing.T) {
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
		Directories: []string{"/dir/one", "/dir/bad", "/dir/three"},
		CacheDir:    "ok/cache",
		ResultsDir:  "ok/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
	if err == nil {
		t.Fatal("Expected error for invalid directory, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected ErrNoReadAccess, got: %v", err)
	}
}

func TestResolveAndValidateArgs_Fails_ResultsDir(t *testing.T) {
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
		Directories: []string{"/ok/source"},
		ResultsDir:  "/bad/results",
	}

	err := r.ResolveAndValidateArgs(args, "/exe")
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
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, CPUs: 0},
			expected: models.ExecutionParams{CPUs: maxCPUs},
		},
		{
			name:     "Negative value defaults to Max",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, CPUs: -5},
			expected: models.ExecutionParams{CPUs: maxCPUs},
		},
		{
			name:     "Valid value within range",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, CPUs: 2},
			expected: models.ExecutionParams{CPUs: 2},
		},
		{
			name:     "Value exceeding Max is capped",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, CPUs: maxCPUs + 1},
			expected: models.ExecutionParams{CPUs: maxCPUs},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// We pass the address of the field from the struct instance
			err := r.ResolveAndValidateArgs(&tt.params, "")
			if err != nil {
				t.Errorf("%s: Some error %d", tt.name, err)
			}
			if tt.params.CPUs != tt.expected.CPUs {
				t.Errorf("Expected %d but got %d", tt.expected.CPUs, tt.params.CPUs)
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
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, BufSize: 0},
			expected: models.ExecutionParams{BufSize: defaultBuf},
		},
		{
			name:     "Negative value defaults",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, BufSize: -1},
			expected: models.ExecutionParams{BufSize: defaultBuf},
		},
		{
			name:     "Specific valid buffer",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, BufSize: 2048},
			expected: models.ExecutionParams{BufSize: 2048},
		},
		{
			name:     "Specific valid buffer, over max",
			params:   models.ExecutionParams{Directories: []string{"/placeholder"}, BufSize: 100000000000},
			expected: models.ExecutionParams{BufSize: maxValue},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// We pass the address of the field from the struct instance
			err := r.ResolveAndValidateArgs(&tt.params, "")
			if err != nil {
				t.Errorf("%s: Some error %d", tt.name, err)
			}
			if tt.params.BufSize != tt.expected.BufSize {
				t.Errorf("Expected %d but got %d", tt.expected.BufSize, tt.params.BufSize)
			}
		})
	}
}
