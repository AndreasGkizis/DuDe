package services

import (
	"DuDe/internal/models"
	"sync"
)

// RemoveUniqueSizeFiles removes files whose size is unique within the
// collection. Such files cannot have duplicates in the same execution and do
// not need to be hashed.
func RemoveUniqueSizeFiles(files *sync.Map) (candidateCount, skippedCount int) {
	sizeCounts := make(map[int64]int)
	files.Range(func(_, value any) bool {
		file := value.(models.FileHash)
		sizeCounts[file.FileSize]++
		return true
	})

	files.Range(func(key, value any) bool {
		file := value.(models.FileHash)
		if sizeCounts[file.FileSize] == 1 {
			files.Delete(key)
			skippedCount++
			return true
		}

		candidateCount++
		return true
	})

	return candidateCount, skippedCount
}

// RemoveUnmatchedSizeFiles removes files whose size is not present in the
// supplied set. It is useful when one collection, such as a coverage target,
// is filtered against sizes discovered in another collection.
// FileSizes returns the set of sizes present in a file collection.
func FileSizes(files *sync.Map) map[int64]struct{} {
	sizes := make(map[int64]struct{})
	files.Range(func(_, value any) bool {
		file := value.(models.FileHash)
		sizes[file.FileSize] = struct{}{}
		return true
	})
	return sizes
}

// CopyFilesWithSizes creates a filtered collection without changing the
// original discovered files.
func CopyFilesWithSizes(files *sync.Map, allowedSizes map[int64]struct{}) *sync.Map {
	filtered := &sync.Map{}
	files.Range(func(key, value any) bool {
		file := value.(models.FileHash)
		if _, allowed := allowedSizes[file.FileSize]; allowed {
			filtered.Store(key, file)
		}
		return true
	})
	return filtered
}

func RemoveUnmatchedSizeFiles(files *sync.Map, allowedSizes map[int64]struct{}) (candidateCount, skippedCount int) {
	files.Range(func(key, value any) bool {
		file := value.(models.FileHash)
		if _, allowed := allowedSizes[file.FileSize]; !allowed {
			files.Delete(key)
			skippedCount++
			return true
		}

		candidateCount++
		return true
	})

	return candidateCount, skippedCount
}
