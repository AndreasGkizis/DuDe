package processing

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
	"context"
	"strings"
	"time"

	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func WalkDir(ctx context.Context, path string, result *sync.Map, pt *visuals.ProgressCounter) {
	defer func() {
		pt.SenderFinished()
	}()

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started walking directory %s files", groupID, path))

	err := filepath.WalkDir(path, storeFilePaths(ctx, result, pt))

	if err != nil {
		// Check if the error was due to user cancellation
		if errors.Is(err, context.Canceled) {
			log.WarnWithFuncName(fmt.Sprintf("Group %d walking cancelled by user.", groupID))
			// Do not treat cancellation as a failure
			return
		}
		log.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished walking directory %s files", groupID, path))
}

func storeFilePaths(ctx context.Context, result *sync.Map, pt *visuals.ProgressCounter) func(path string, dirEntry fs.DirEntry, err error) error {
	return func(path string, dirEntry fs.DirEntry, err error) error {

		// --- 1. Cancellation Check ---
		select {
		case <-ctx.Done():
			// If the context is done, stop the walk immediately.
			// Returning the context error will cause WalkDir to stop
			// and return context.Canceled.
			log.WarnWithFuncName(fmt.Sprintf("Stopping file walk at %s due to context cancellation.", path))
			return ctx.Err() // returns context.Canceled
		default:
			// Continue if not cancelled
		}

		if err != nil {
			reportSkippedWalkEntry(ctx, pt, path, err)
			if dirEntry != nil && dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if dirEntry == nil {
			err := fmt.Errorf("directory entry is unavailable")
			reportSkippedWalkEntry(ctx, pt, path, err)
			return nil
		}

		if !dirEntry.IsDir() {
			info, err := dirEntry.Info()
			if err != nil {
				reportSkippedWalkEntry(ctx, pt, path, err)
				return nil
			}

			result.Store(path, models.FileHash{
				FileName: dirEntry.Name(),
				FilePath: path,
				FileSize: info.Size(),
				ModTime:  info.ModTime().Format(common.TimeFrmt),
			})

			pt.Channel <- 1
		}
		return nil
	}
}

func reportSkippedWalkEntry(ctx context.Context, pt *visuals.ProgressCounter, path string, err error) {
	message := fmt.Sprintf("Skipped %s: %v", path, err)
	if errors.Is(err, os.ErrPermission) || os.IsNotExist(err) {
		log.WarnWithFuncName(message)
	} else {
		log.ErrorWithFuncName(message)
	}
	pt.Reporter.LogDetailedStatus(ctx, message)
}

func FilterHashCandidatesBySize(files *sync.Map) (candidateCount, skippedCount int) {
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

func SaveResultsAsCSV(data *sync.Map, fulldir string) error {
	flattened_data := GetFlattened(data)
	log.InfoWithFuncName(fmt.Sprintf("Number of duplicates found: %d", len(flattened_data)))
	log.InfoWithFuncName(fmt.Sprintf("Creating results file in: %s", fulldir))

	if len(flattened_data) == 0 {
		log.WarnWithFuncName("No results file produced, 0 duplicates found")
		return nil
	}

	file, err := createResultFile(fulldir)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Comma = GetDelimiterForOS()

	// Write the UTF-8 BOM bytes at the very beginning of the file to force stupid excel to recognise the encoding.
	_, err = file.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		return fmt.Errorf("failed to write UTF-8 BOM: %v", err)
	}

	err = writer.Write(common.ResultsHeader)
	if err != nil {
		return err
	}

	for _, entry := range flattened_data {
		err = writer.Write([]string{
			entry.Filename,
			entry.FullPath,
			entry.DuplicateFilename,
			entry.DuplicateFullPath,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func ResultsFileExist(path string) bool {

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return false
	}

	for _, entry := range entries {
		// Skip directories to ensure we only process files
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if file starts with "results" and ends with ".csv"
		if strings.HasPrefix(name, "results") && strings.HasSuffix(name, ".csv") {
			fmt.Printf("Found matching file: %s\n", name)
			return true
		}
	}
	return false
}

func GetDelimiterForOS() rune {
	var delimiter rune
	if runtime.GOOS == "windows" {
		delimiter = ';'
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	} else {
		delimiter = ',' // Default for Linux, macOS, etc.
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	}
	return delimiter
}

func createResultFile(fulldir string) (*os.File, error) {

	datetime := time.Now()
	filename := fmt.Sprint(common.Results_file_name, datetime.Format("_2006_01_02_15_04_05"), ".", common.Results_file_extension)
	filepath := filepath.Join(fulldir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Results file with name [%s] already exists", filename))
		return nil, err
	}
	return file, nil
}
