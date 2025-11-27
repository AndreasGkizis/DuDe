package handlers

import (
	common "DuDe/internal/common"
	"DuDe/internal/models"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Define specific validation error types
var (
	ErrPathNotDirectory = errors.New("path is not a directory")
	ErrPathNotExists    = errors.New("path does not exist")
	ErrNoReadAccess     = errors.New("no read access")
	ErrNoWriteAccess    = errors.New("no write access")
	ErrPathIsDirectory  = errors.New("path is a directory but should be a file")
)

// ResolveAndValidateArgs applies necessary defaults and then validates all paths.
func ResolveAndValidateArgs(args *models.ExecutionParams) error {

	// --------------------------------------------------
	// 1. APPLY DEFAULTS (Path Resolution)
	// --------------------------------------------------
	executableDir := common.GetExecutableDir()

	// CacheDir Default
	if args.CacheDir == "" || args.CacheDir == common.Def {
		args.CacheDir = executableDir
	}

	// ResultsDir Default
	if args.ResultsDir == "" || args.ResultsDir == common.Def {
		args.ResultsDir = executableDir
	}

	// --------------------------------------------------
	// 2. VALIDATION (using resolved paths)
	// --------------------------------------------------

	// 2a. SourceDir (Must exist and be readable)
	if args.SourceDir == "" {
		return errors.New("source directory cannot be empty after resolution")
	}
	if err := validatePath(args.SourceDir, true, false); err != nil {
		return fmt.Errorf("SourceDir validation failed: %w", err)
	}

	// 2b. TargetDir (Optional, but if present, must exist and be readable)
	if args.TargetDir != "" {
		if err := validatePath(args.TargetDir, true, false); err != nil {
			return fmt.Errorf("TargetDir validation failed: %w", err)
		}
	}

	// 2c. CacheDir (Must be writable, now guaranteed to be non-empty)
	if err := validatePath(args.CacheDir, false, true); err != nil {

		// If the path does not exist, check if its parent is writable
		if errors.Is(err, ErrPathNotExists) {
			parentDir := filepath.Dir(args.CacheDir)
			if err := checkWriteAccess(parentDir, fmt.Errorf("parent directory %s does not allow writing", parentDir)); err != nil {
				return fmt.Errorf("CacheDir parent write check failed: %w", err)
			}
			// If parent is writable, validation passes, as we can create the directory later.
		} else {
			// Handle existing path that is a file, or other permission issues on an existing directory
			return fmt.Errorf("CacheDir validation failed: %w", err)
		}
	}

	// 2d. ResultsDir (Must be writable, now guaranteed to be non-empty)
	if err := validatePath(args.ResultsDir, false, true); err != nil {

		// If the path does not exist, check if its parent is writable
		if errors.Is(err, ErrPathNotExists) {
			parentDir := filepath.Dir(args.ResultsDir)
			if err := checkWriteAccess(parentDir, fmt.Errorf("parent directory %s does not allow writing", parentDir)); err != nil {
				return fmt.Errorf("ResultsDir parent write check failed: %w", err)
			}
			// If parent is writable, validation passes, as we can create the directory later.
		} else {
			// Handle existing path that is a file, or other permission issues on an existing directory
			return fmt.Errorf("ResultsDir validation failed: %w", err)
		}
	}

	return nil
}

// ValidatePath checks path existence, type, and required permissions.
func validatePath(path string, needsRead, needsWrite bool) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	info, err := os.Stat(path)

	// --- Check 1: Existence ---
	if os.IsNotExist(err) {
		// If the path doesn't exist, we must only proceed if it's a new path
		// that requires writing (like CacheDir or ResultsDir).
		if needsWrite {
			// For new paths, we only need to check write access on the parent directory.
			// Note: This logic is now mostly handled outside for file paths (ResultsDir).
			return nil // Exit validation early, as we will check parent directory separately.
		}
		return fmt.Errorf("%w: %s", ErrPathNotExists, path)
	}

	if err != nil {
		return err // OS-level error beyond existence/permission
	}
	isDir := info.IsDir()
	// --- Check 2: Path Type Verification (Must be a directory) ---
	if !isDir {
		// Path exists but is not a directory (i.e., it's a file).
		return fmt.Errorf("%w: %s", ErrPathNotDirectory, path)
	}

	if needsRead {
		// Check read access for existing directory
		if err := checkReadAccess(path); err != nil {
			return fmt.Errorf("read check failed on %s: %w", path, err)
		}
	}
	if needsWrite {
		// Check write access for existing directory
		if err := checkWriteAccess(path, fmt.Errorf("%w: %s", ErrNoWriteAccess, path)); err != nil {
			return fmt.Errorf("write check failed on %s: %w", path, err)
		}
	}

	return nil
}

// checkReadAccess attempts to open the directory to ensure reading is possible.
func checkReadAccess(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrNoReadAccess, path)
	}
	f.Close()
	return nil
}

// checkWriteAccess attempts to create a temporary file inside the path to ensure writing is possible.
func checkWriteAccess(path string, defaultErr error) error {
	// Attempt to create a unique temporary file inside the directory
	tempFile := filepath.Join(path, fmt.Sprintf(".temp_dude_test_%d", os.Getpid()))

	// Use os.O_CREATE and os.O_WRONLY to check write permission
	f, err := os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)

	if err != nil {
		// If we can't create the file, we likely don't have write permissions.
		return defaultErr
	}

	// Clean up the temporary file immediately
	f.Close()
	os.Remove(tempFile)

	return nil
}
