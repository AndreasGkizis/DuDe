package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func GetExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err) // Handle error appropriately in production code
	}
	return filepath.Dir(exePath)
}

// GetSafeResultsDir returns a writable directory for saving results.
// On macOS, app bundles are not writable, so we use the user's Documents folder.
// On other platforms, we use the executable directory.
func GetSafeResultsDir(platform string) string {
	if platform == "darwin" {
		// On macOS, use Documents folder instead of app bundle
		homeDir, err := os.UserHomeDir()
		if err == nil {
			documentsDir := filepath.Join(homeDir, "Documents", "DuDe")
			// Create the directory if it doesn't exist
			os.MkdirAll(documentsDir, 0755)
			return documentsDir
		}
	}
	// Fallback to executable directory for Windows/Linux
	return GetExecutableDir()
}

func GetFileDir(path string) string {
	return filepath.Dir(path)
}

func GetOpenDirectoryFunc(path, platform string) (*exec.Cmd, error) {
	switch platform {
	case "windows":
		return exec.Command("explorer", path), nil
	case "darwin":
		return exec.Command("open", path), nil // macOS: uses 'open' command
	case "linux":
		return exec.Command("xdg-open", path), nil // Linux: uses 'xdg-open'
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func LenSyncMap(m *sync.Map) int {
	var count int
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
