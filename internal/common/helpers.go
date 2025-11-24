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

func ConvertSyncMapToMap(sMap *sync.Map) map[interface{}]interface{} {
	if sMap == nil {
		return nil // Or return an empty map, depending on desired behavior
	}

	resultMap := make(map[interface{}]interface{})

	// Iterate over the sync.Map and store its contents in the regular map.
	// The Range method is thread-safe.
	sMap.Range(func(key, value interface{}) bool {
		resultMap[key] = value
		return true // Continue iteration
	})

	return resultMap
}
