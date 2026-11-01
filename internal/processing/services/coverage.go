package services

import (
	"DuDe/internal/models"
	"context"
	"fmt"
	"os"
	"sort"
)

// MatchCoverage checks every source file for an equivalent file in the target
// collection. Target candidates are never consumed, so one target file may
// cover any number of identical source files.
func MatchCoverage(ctx context.Context, sourceFiles, targetFiles []models.FileHash, paranoid bool) (models.MissingContentResult, error) {
	result := models.MissingContentResult{
		TotalSourceFiles: len(sourceFiles),
		MissingFiles:     make([]models.MissingSourceFile, 0),
	}

	targetBySize := make(map[int64][]models.FileHash)
	for _, target := range targetFiles {
		targetBySize[target.FileSize] = append(targetBySize[target.FileSize], target)
	}
	for size := range targetBySize {
		sort.Slice(targetBySize[size], func(i, j int) bool {
			return targetBySize[size][i].FilePath < targetBySize[size][j].FilePath
		})
	}

	for _, source := range sourceFiles {
		if err := ctx.Err(); err != nil {
			return models.MissingContentResult{}, err
		}

		candidates := targetBySize[source.FileSize]
		covered, err := sourceCovered(ctx, source, candidates, paranoid)
		if err != nil {
			return models.MissingContentResult{}, fmt.Errorf("check source file %q: %w", source.FilePath, err)
		}
		if !covered {
			result.MissingFiles = append(result.MissingFiles, models.MissingSourceFile{
				Filename: source.FileName,
				FullPath: source.FilePath,
				Size:     source.FileSize,
			})
		}
	}

	sort.Slice(result.MissingFiles, func(i, j int) bool {
		return result.MissingFiles[i].FullPath < result.MissingFiles[j].FullPath
	})
	return result, nil
}

func sourceCovered(ctx context.Context, source models.FileHash, candidates []models.FileHash, paranoid bool) (bool, error) {
	if len(candidates) == 0 {
		return false, nil
	}

	if !paranoid {
		for _, candidate := range candidates {
			if source.Hash != "" && source.Hash == candidate.Hash {
				return true, nil
			}
		}
		return false, nil
	}

	primary, err := os.Open(source.FilePath)
	if err != nil {
		return false, err
	}
	defer primary.Close()

	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		matched, err := FilesMatch(ctx, primary, source, candidate, true)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
