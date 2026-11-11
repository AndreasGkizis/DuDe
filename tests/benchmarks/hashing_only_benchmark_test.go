package benchmarks

import (
	"context"
	"runtime"
	"testing"

	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"

	log "DuDe/internal/common/logger"
)

// BenchmarkCreateHashes measures DuDe's production candidate-filtering and
// hashing pipeline with caching disabled. Run it before and after changing the
// hashing algorithm; the workload and worker configuration stay identical.
func BenchmarkCreateHashes(b *testing.B) {
	log.Initialize(false)

	scenarios := []struct {
		name  string
		sizes []int64
	}{
		{
			name:  "128x64KiB_same_size",
			sizes: repeatedSizes(128, 64*1024),
		},
		{
			name:  "128_mixed_sizes_50pct_candidates",
			sizes: sizesWithSharedPercentage(128, 50),
		},
		{
			name:  "32x4MiB_same_size",
			sizes: repeatedSizes(32, 4*1024*1024),
		},
	}

	for _, scenario := range scenarios {
		files := createBenchmarkFiles(b, scenario.sizes)

		b.Run(scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(totalSize(files))

			for range b.N {
				b.StopTimer()
				sourceFiles := sourceMap(files)
				hashMemory := make(map[string]models.FileHash)
				memoryManager := services.NewMemoryManager(&models.ExecutionParams{}, len(files), 1)
				tracker := visuals.NewProgressTracker(context.Background(), reporting.NoOpReporter{}, "Hashing")
				sink := services.NewDiagnosticSink()
				b.StartTimer()

				// This is part of the production duplicate flow. It removes
				// files whose sizes cannot have duplicates before hashing.
				services.RemoveUniqueSizeFiles(sourceFiles)

				err := services.CreateHashes(
					context.Background(),
					sourceFiles,
					runtime.NumCPU(),
					tracker,
					memoryManager,
					&hashMemory,
					sink,
					func(services.DiagnosticEvent) services.ErrorDisposition {
						return services.ErrorDiagnostic
					},
					models.DiagnosticScopeDuplicate,
				)
				if err != nil {
					b.Fatalf("create hashes: %v", err)
				}
				sink.Close()
			}
		})
	}
}

func totalSize(files []benchmarkFile) int64 {
	var total int64
	for _, file := range files {
		total += file.size
	}
	return total
}
