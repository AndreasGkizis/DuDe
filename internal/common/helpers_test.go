package common

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetExecutableDirReturnsExecutableParent(t *testing.T) {
	got, err := GetExecutableDir()
	if err != nil {
		t.Fatalf("GetExecutableDir() returned an error: %v", err)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned an error: %v", err)
	}
	want := filepath.Dir(executable)
	if got != want {
		t.Fatalf("GetExecutableDir() = %q, want %q", got, want)
	}
}

func TestResolveResultsDirDarwinCreatesDocumentsDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir does not use HOME on Windows")
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	got, err := ResolveResultsDir("darwin")
	if err != nil {
		t.Fatalf("ResolveResultsDir() returned an error: %v", err)
	}

	want := filepath.Join(homeDir, "Documents", "DuDe")
	if got != want {
		t.Fatalf("ResolveResultsDir() = %q, want %q", got, want)
	}

	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("created results directory cannot be inspected: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("created results path %q is not a directory", want)
	}
}

func TestResolveResultsDirDarwinReturnsDirectoryCreationError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir does not use HOME on Windows")
	}

	homePath := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homePath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create HOME fixture: %v", err)
	}
	t.Setenv("HOME", homePath)

	_, err := ResolveResultsDir("darwin")
	if err == nil {
		t.Fatal("ResolveResultsDir() returned nil error for an unusable home directory")
	}
}

func TestResolveResultsDirNonDarwinUsesExecutableDirectory(t *testing.T) {
	got, err := ResolveResultsDir("linux")
	if err != nil {
		t.Fatalf("ResolveResultsDir() returned an error: %v", err)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned an error: %v", err)
	}
	want := filepath.Dir(executable)
	if got != want {
		t.Fatalf("ResolveResultsDir() = %q, want %q", got, want)
	}
}
