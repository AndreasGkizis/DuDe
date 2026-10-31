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
