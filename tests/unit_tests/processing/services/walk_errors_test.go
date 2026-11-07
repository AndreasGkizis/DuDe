package unit_tests

import (
	"DuDe/internal/models"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWalkDirMissingRootDoesNotPanic(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "removed")
	files, diagnostics, err := discoverFiles(context.Background(), []string{missingRoot})
	if err != nil {
		t.Fatalf("expected missing root to be recoverable, got %v", err)
	}
	count := 0
	files.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatal("expected a missing root directory to produce no files")
	}
	if len(diagnostics) != 1 || diagnostics[0].Path != missingRoot {
		t.Fatalf("expected a diagnostic for the missing root, got %#v", diagnostics)
	}
}

func TestWalkDirReportsUnreadableDirectoryAndContinuesWithSiblings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are required for this test")
	}

	root := t.TempDir()
	blockedDirectory := filepath.Join(root, "blocked")
	if err := os.Mkdir(blockedDirectory, 0o700); err != nil {
		t.Fatalf("create blocked directory: %v", err)
	}
	blockedFile := filepath.Join(blockedDirectory, "blocked.txt")
	if err := os.WriteFile(blockedFile, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("write blocked file: %v", err)
	}
	accessibleFile := filepath.Join(root, "z-accessible.txt")
	if err := os.WriteFile(accessibleFile, []byte("accessible"), 0o600); err != nil {
		t.Fatalf("write accessible file: %v", err)
	}

	if err := os.Chmod(blockedDirectory, 0); err != nil {
		t.Fatalf("remove directory permissions: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDirectory, 0o700) })
	if _, err := os.ReadDir(blockedDirectory); err == nil {
		t.Skip("current user can still read permissionless directories")
	}

	files, diagnostics, err := discoverFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatalf("expected unreadable directory to be recoverable, got %v", err)
	}
	if _, exists := files.Load(accessibleFile); !exists {
		t.Fatal("expected scanning to continue with accessible sibling files")
	}
	if _, exists := files.Load(blockedFile); exists {
		t.Fatal("expected files below the unreadable directory to be skipped")
	}
	if len(diagnostics) == 0 || diagnostics[0].Scope != models.DiagnosticScopeDuplicate {
		t.Fatalf("expected an unreadable-directory diagnostic, got %#v", diagnostics)
	}
}
