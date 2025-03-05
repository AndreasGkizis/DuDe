package common

import (
	"os"
	"path/filepath"
)

func GetExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err) // Handle error appropriately in production code
	}
	return filepath.Dir(exePath)
}
