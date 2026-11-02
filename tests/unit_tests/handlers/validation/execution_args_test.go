package unit_test

import (
	val "DuDe/internal/handlers/validation"
	"DuDe/internal/models"
	"errors"
	"runtime"
	"testing"
)

func TestResolveExecutionArgsAppliesSharedDefaults(t *testing.T) {
	args := models.ExecutionParams{}

	val.ResolveExecutionArgs(&args, "/fallback")

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

func TestResolveExecutionArgsPreservesProvidedDirectories(t *testing.T) {
	args := models.ExecutionParams{
		CacheDir:   "/cache",
		ResultsDir: "/results",
	}

	val.ResolveExecutionArgs(&args, "/fallback")

	if args.CacheDir != "/cache" {
		t.Fatalf("CacheDir = %q, want /cache", args.CacheDir)
	}
	if args.ResultsDir != "/results" {
		t.Fatalf("ResultsDir = %q, want /results", args.ResultsDir)
	}
}

func TestResolveExecutionArgsResolvesWorkerCount(t *testing.T) {
	maxWorkers := runtime.GOMAXPROCS(0)
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero defaults", input: 0, want: maxWorkers},
		{name: "negative defaults", input: -1, want: maxWorkers},
		{name: "valid preserved", input: 1, want: 1},
		{name: "over maximum capped", input: maxWorkers + 1, want: maxWorkers},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := models.ExecutionParams{WorkerCount: test.input}
			val.ResolveExecutionArgs(&args, "/fallback")
			if args.WorkerCount != test.want {
				t.Fatalf("WorkerCount = %d, want %d", args.WorkerCount, test.want)
			}
		})
	}
}

func TestResolveExecutionArgsResolvesBufferSize(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero defaults", input: 0, want: 1024},
		{name: "negative defaults", input: -1, want: 1024},
		{name: "valid preserved", input: 2048, want: 2048},
		{name: "over maximum capped", input: 1048577, want: 1048576},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := models.ExecutionParams{BufferSizeBytes: test.input}
			val.ResolveExecutionArgs(&args, "/fallback")
			if args.BufferSizeBytes != test.want {
				t.Fatalf("BufferSizeBytes = %d, want %d", args.BufferSizeBytes, test.want)
			}
		})
	}
}

func TestValidateSharedExecutionArgsChecksResultsDirectory(t *testing.T) {
	var validatedPath string
	paths := val.MockValidator{
		WritableDirFunc: func(path string) error {
			validatedPath = path
			return nil
		},
		ReadableDirFunc: func(path string) error {
			t.Fatalf("shared validation unexpectedly checked readable path %q", path)
			return nil
		},
	}
	args := models.ExecutionParams{ResultsDir: "/results"}

	err := val.ValidateSharedExecutionArgs(&args, paths)

	if err != nil {
		t.Fatalf("validate shared args: %v", err)
	}
	if validatedPath != "/results" {
		t.Fatalf("validated path = %q, want /results", validatedPath)
	}
}

func TestValidateSharedExecutionArgsWrapsResultsDirectoryError(t *testing.T) {
	expected := errors.New("permission denied")
	paths := val.MockValidator{
		WritableDirFunc: func(string) error { return expected },
	}
	args := models.ExecutionParams{ResultsDir: "/results"}

	err := val.ValidateSharedExecutionArgs(&args, paths)

	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}
