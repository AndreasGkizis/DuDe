package benchmarks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"DuDe/internal/processing"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
)

type benchmarkFile struct {
	path string
	size int64
}

type benchmarkScenario struct {
	name  string
	sizes []int64
}

const benchmarkFileSize int64 = 64 * 1024

func BenchmarkHashAndGroupPipeline(b *testing.B) {
	log.Initialize(false)

	sharedSizePercentages := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	scenarios := make([]benchmarkScenario, 0, len(sharedSizePercentages))
	for _, percentage := range sharedSizePercentages {
		scenarios = append(scenarios, benchmarkScenario{
			name:  fmt.Sprintf("%03d_percent_same_size", percentage),
			sizes: sizesWithSharedPercentage(128, percentage),
		})
	}

	for _, scenario := range scenarios {
		files := createBenchmarkFiles(b, scenario.sizes)

		b.Run(scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				b.StopTimer()
				sourceFiles := sourceMap(files)
				hashMemory := make(map[string]models.FileHash)
				memoryManager := processing.NewMemoryManager(&models.ExecutionParams{}, len(files), 1)
				hashTracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Hashing")
				findTracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Finding")
				errChan := make(chan error, len(files))
				failedCount := 0
				b.StartTimer()

				processing.FilterHashCandidatesBySize(sourceFiles)
				err := processing.CreateHashes(
					context.Background(),
					sourceFiles,
					runtime.NumCPU(),
					hashTracker,
					memoryManager,
					&hashMemory,
					&failedCount,
					errChan,
				)
				if err != nil {
					b.Fatalf("create hashes: %v", err)
				}

				processing.FindDuplicatesInMap(context.Background(), sourceFiles, findTracker)

				select {
				case err := <-errChan:
					b.Fatalf("pipeline error: %v", err)
				default:
				}
			}
		})
	}
}

func sourceMap(files []benchmarkFile) *sync.Map {
	sourceFiles := &sync.Map{}
	for _, file := range files {
		sourceFiles.Store(file.path, models.FileHash{
			FilePath: file.path,
			FileSize: file.size,
		})
	}
	return sourceFiles
}

func createBenchmarkFiles(b *testing.B, sizes []int64) []benchmarkFile {
	b.Helper()

	directory := b.TempDir()
	files := make([]benchmarkFile, 0, len(sizes))
	for index, size := range sizes {
		contents := make([]byte, int(size))
		for offset := range contents {
			contents[offset] = byte((index + offset) % 251)
		}

		path := filepath.Join(directory, fmt.Sprintf("file-%04d.bin", index))
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			b.Fatalf("create benchmark file: %v", err)
		}
		files = append(files, benchmarkFile{path: path, size: size})
	}
	return files
}

func sizesWithSharedPercentage(total, sharedSizePercentage int) []int64 {
	if sharedSizePercentage < 0 || sharedSizePercentage > 100 {
		panic("shared size percentage must be between 0 and 100")
	}

	sharedSizeCount := total * sharedSizePercentage / 100
	uniqueSizeCount := total - sharedSizeCount
	sizes := uniqueSizes(uniqueSizeCount)
	return append(sizes, repeatedSizes(sharedSizeCount, benchmarkFileSize)...)
}

func uniqueSizes(count int) []int64 {
	sizes := make([]int64, count)
	for index := range sizes {
		offset := int64(index - count/2)
		if offset >= 0 {
			offset++
		}
		sizes[index] = benchmarkFileSize + offset
	}
	return sizes
}

func repeatedSizes(count int, size int64) []int64 {
	sizes := make([]int64, count)
	for index := range sizes {
		sizes[index] = size
	}
	return sizes
}
