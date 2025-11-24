package processing

import (
	com "DuDe-wails/internal/common"
	logger "DuDe-wails/internal/common/logger"
	models "DuDe-wails/internal/models"
	visuals "DuDe-wails/internal/visuals"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func CreateHashes(sourceFiles *sync.Map, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash, failedCount *int, errChan chan error) error {
	numFilesToHash := com.LenSyncMap(sourceFiles)
	if numFilesToHash == 0 {
		return nil // Nothing to hash
	}

	groupID := rand.Uint32()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d started hashing %d files with %d workers", groupID, numFilesToHash, maxWorkers))
	pt.AddTotal(int64(numFilesToHash))

	var wg sync.WaitGroup

	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size

	sourceFiles.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(path string, val models.FileHash) {
			defer wg.Done()
			var hash string
			var err error

			currentFilePath := key.(string)

			// Acquire a slot
			sem <- struct{}{}
			defer func() { <-sem }() // Release the slot

			currentFileDiskStats, err := os.Stat(val.FilePath)
			if err != nil {
				errChan <- err
				sourceFiles.Delete(val.FilePath)
				pt.DecrementFromTotal() // remove for progress bar
				return                  // stop this iteration
			}

			currentFileDiskSize := currentFileDiskStats.Size()
			currentFileDiskModTime := currentFileDiskStats.ModTime().Format(time.RFC3339)

			memoryOfFile, memoryExists := (*memory)[currentFilePath]

			fileHasChangedOnDisk := memoryOfFile.FileSize != currentFileDiskSize || memoryOfFile.ModTime != currentFileDiskModTime

			fileNeedsReHashing := !memoryExists || fileHasChangedOnDisk

			if fileNeedsReHashing {
				hash, err = calculateMD5Hash(val)

				if err != nil {
					sourceFiles.Delete(val.FilePath)
					pt.DecrementFromTotal() // remove for progress bar
					errChan <- err
					return // stop this iteration
				}

				newMem := models.FileHash{
					FileName: filepath.Base(path),
					FilePath: path,
					Hash:     hash,
					FileSize: currentFileDiskSize,
					ModTime:  currentFileDiskModTime,
				}

				sourceFiles.Store(path, newMem)
				sendWithRetry(mm.Channel, newMem, 500*time.Millisecond, 5*time.Second, failedCount)

			} else {
				sourceFiles.Store(path, memoryOfFile)
			}

			// safeResend(mm.Channel, newMem, 500*time.Microsecond)

			pt.Increment()
		}(key.(string), value.(models.FileHash))
		return true
	})

	wg.Wait()
	mm.SenderFinished()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d finished hashing %d files with %d workers", groupID, int64(numFilesToHash), maxWorkers))

	close(sem)

	return nil // Return the first error that occurred, or nil if none
}

func EnsureDuplicates(input *sync.Map, pt *visuals.ProgressTracker, maxWorkers int) {
	num := 0

	input.Range(func(key, value any) bool {
		num += len(value.(models.FileHash).DuplicatesFound)
		return true
	})

	if num == 0 {
		logger.WarnWithFuncName("No duplicates to ensure")
	}

	pt.AddTotal(int64(num))

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	input.Range(func(itemHash, item any) bool {
		wg.Add(1)
		go func(itemHash string, item models.FileHash) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if len(item.DuplicatesFound) == 0 {
				return
			}

			mainFile, err := os.Open(item.FilePath)

			if err != nil {
				logger.WarnWithFuncName(fmt.Sprintf("Error opening file %s : %v. skipping..", item.FilePath, err))
				return
			}

			defer mainFile.Close()

			for dupIndex := 0; dupIndex < len(item.DuplicatesFound); {
				dup := item.DuplicatesFound[dupIndex]

				eq, err := filesEqual(mainFile, dup.FilePath)

				if err != nil {
					logger.WarnWithFuncName(fmt.Sprintf("Error comparing files %s and %s: %v. Considering as equal.", item.FilePath, dup.FilePath, err))
					eq = true
				}

				if !eq {
					item.DuplicatesFound = append(item.DuplicatesFound[:dupIndex], item.DuplicatesFound[dupIndex+1:]...)
					if len(item.DuplicatesFound) == 0 {
						input.Delete(itemHash)
					}
				} else {
					dupIndex++
				}
				// reset readers
				_, _ = mainFile.Seek(0, io.SeekStart)
				pt.Increment()
			}
		}(itemHash.(string), item.(models.FileHash))
		return true
	})
}

func filesEqual(file1 *os.File, path2 string) (bool, error) {
	file2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("error opening duplicate file: %w", err)
	}
	defer file2.Close()

	const chunkSize = 4096
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if err1 != nil && err1 != io.EOF || err2 != nil && err2 != io.EOF {
			return false, fmt.Errorf("read error: %w", errors.Join(err1, err2))
		}

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}
	}

	// Reset both files for potential reuse
	file1.Seek(0, io.SeekStart)
	return true, nil
}

func calculateMD5Hash(file models.FileHash) (string, error) {
	hasherMD5 := md5.New()

	f, err := os.Open(file.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			logger.WarnWithFuncName(fmt.Sprintf("Skipping file: %s, reason: %s", file.FilePath, err.Error()))
			return "", err
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			logger.ErrorWithFuncName(err.Error())
		}
	}()

	io.Copy(hasherMD5, f)
	// add blob suffix for uniquness
	return fmt.Sprintf("%x", hasherMD5.Sum(nil)), nil
}

func FindDuplicatesInMap(fileHashes *sync.Map, tracker *visuals.ProgressTracker) {
	timer := time.Now()
	initialCount := com.LenSyncMap(fileHashes)

	groupID := rand.Uint32()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d started for source folder with %d files", groupID, initialCount))

	hashCounts := make(map[string]int)
	hashPaths := make(map[string][]models.FileHash)

	fileHashes.Range(func(_, value any) bool {
		hash := value.(models.FileHash).Hash

		hashCounts[hash]++
		hashPaths[hash] = append(hashPaths[hash], value.(models.FileHash))
		return true
	})

	totalGroups := len(hashPaths)
	tracker.AddTotal(int64(totalGroups))

	fileHashes.Clear()

	for hash, files := range hashPaths {
		if len(files) == 1 {
			delete(hashPaths, hash)
			tracker.Increment()
		} else {
			file := files[0] // smallest name?
			dups := []models.FileHash{}
			for i := 1; i < len(files); i++ {
				dups = append(dups, files[i])
			}
			file.DuplicatesFound = dups
			fileHashes.Store(file.Hash, file)
			tracker.Increment()
		}
	}

	length2 := com.LenSyncMap(fileHashes)
	logger.InfoWithFuncName(fmt.Sprintf("%d", length2))

	logger.InfoWithFuncName(fmt.Sprintf("Group %d finished, took : %s .source folder with %d files", groupID, time.Since(timer), initialCount))
}

func GetDuplicates(input *sync.Map) map[string]models.FileHash {

	seen := make(map[string]models.FileHash)
	result := make(map[string]models.FileHash)

	input.Range(func(key, value any) bool {

		hash := value.(models.FileHash).Hash
		path := key.(string)

		_, ok := seen[path]
		if !ok {
			seen[path] = value.(models.FileHash)
		} else {
			result[hash] = value.(models.FileHash)
		}

		return true
	})
	return result
}

func GetFlattened(input *sync.Map) []models.ResultEntry {
	result := make([]models.ResultEntry, 0)

	separatorEntry := models.ResultEntry{
		Filename:          com.ResultsFileSeperator,
		FullPath:          com.ResultsFileSeperator,
		DuplicateFilename: com.ResultsFileSeperator,
		DuplicateFullPath: com.ResultsFileSeperator}

	input.Range(func(key, value any) bool {
		val := value.(models.FileHash)
		for _, dup := range val.DuplicatesFound {
			a := models.ResultEntry{
				Filename:          val.FileName,
				FullPath:          val.FilePath,
				DuplicateFilename: dup.FileName,
				DuplicateFullPath: dup.FilePath,
			}
			result = append(result, a)
		}
		result = append(result, separatorEntry)

		return true
	})
	return result
}

func sendWithRetry(ch chan models.FileHash, value models.FileHash, baseDelay, maxRetryDelay time.Duration, failedCount *int) error {
	retryDelay := baseDelay
	for {
		select {
		case ch <- value:
			// common.Logger.Warnf("\nData Sent! : %v", value.FileName)
			return nil
		default:
			(*failedCount)++
			// common.Logger.Warnf("\nFailed to send data, retrying in %v: %v", retryDelay, value.FileName)
			time.Sleep(retryDelay)
			retryDelay *= 2
			if retryDelay > maxRetryDelay {
				retryDelay = maxRetryDelay
			}
		}
	}
}
