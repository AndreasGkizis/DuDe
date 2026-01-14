package common

import (
	"os"
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
