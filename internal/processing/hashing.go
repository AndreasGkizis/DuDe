package processing

import (
	com "DuDe/internal/common"
	logger "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
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
	len := com.LenSyncMap(sourceFiles)
	groupID := rand.Uint32()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d started hashing %d files with %d workers", groupID, len, maxWorkers))
	pt.AddTotal(int64(len))

	var wg sync.WaitGroup

	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size
	sourceFiles.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(path string, val models.FileHash) {
			defer wg.Done()
			var hash string
			var err error

			// Acquire a slot
			sem <- struct{}{}
			defer func() { <-sem }() // Release the slot

			stats, err := os.Stat(val.FilePath)
			if err != nil {
				errChan <- err
				sourceFiles.Delete(val.FilePath)
				pt.DecrementFromTotal() // remove for progress bar
				return                  // stop this iteration
			}

			curSize := stats.Size()
			curModTime := stats.ModTime().Format(time.RFC3339)

			var memoryOfFile models.FileHash
			memoryOfFilePath := key.(string)
			memoryOfValue, exists := (*memory)[memoryOfFilePath]
			if exists {
				memoryOfFile = memoryOfValue
			}

			needsReHashing := !exists || (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if needsReHashing {
				hash, err = calculateMD5Hash(val)

				if err != nil {
					sourceFiles.Delete(val.FilePath)
					pt.DecrementFromTotal() // remove for progress bar
					errChan <- err
					return // stop this iteration
				}
			} else {
				hash = memoryOfFile.Hash
			}

			newMem := models.FileHash{
				FileName: filepath.Base(path),
				FilePath: path,
				Hash:     hash,
				FileSize: curSize,
				ModTime:  curModTime,
			}

			sourceFiles.Store(path, newMem)

			// safeResend(mm.Channel, newMem, 500*time.Microsecond)
			sendWithRetry(mm.Channel, newMem, 500*time.Millisecond, 5*time.Second, failedCount)

			pt.Increment()
		}(key.(string), value.(models.FileHash))
		return true
	})

	wg.Wait()
	mm.SenderFinished()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d finished hashing %d files with %d workers", groupID, int64(len), maxWorkers))

	close(sem)

	return nil // Return the first error that occurred, or nil if none
}

func EnsureDuplicates(input map[string]models.FileHash, pt *visuals.ProgressTracker, maxWorkers int) (map[string]models.FileHash, error) {
	num := 0
	for _, v := range input {
		num += len(v.DuplicatesFound)
	}
	if num == 0 {
		logger.WarnWithFuncName("No duplicates to ensure")
		return input, nil
	}
	pt.AddTotal(int64(num))

	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	sem := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	var once sync.Once

	for itemHash, item := range input {
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
				once.Do(func() { errChan <- fmt.Errorf("error opening main file: %w", err) })
				return
			}

			defer mainFile.Close()

			for dupIndex := 0; dupIndex < len(item.DuplicatesFound); {
				dup := item.DuplicatesFound[dupIndex]

				eq, err := filesEqual(mainFile, dup.FilePath)

				if err != nil {
					once.Do(func() { errChan <- err })
					return
				}

				if !eq {
					mu.Lock()
					item.DuplicatesFound = append(item.DuplicatesFound[:dupIndex], item.DuplicatesFound[dupIndex+1:]...)
					if len(item.DuplicatesFound) == 0 {
						delete(input, itemHash)
					}
					mu.Unlock()
				} else {
					dupIndex++
				}
				// reset readers
				_, _ = mainFile.Seek(0, io.SeekStart)
				pt.Increment()
			}
		}(itemHash, item)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()
	if err := <-errChan; err != nil {
		return nil, err
	}
	return input, nil
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
	return fmt.Sprintf("%x", hasherMD5.Sum(nil)), nil
}

func FindDuplicatesInMap(fileHashes *sync.Map) {
	// Map to count occurrences of each hash
	hashCounts := make(map[string]int)
	// Map to store paths for each hash
	hashPaths := make(map[string][]models.FileHash)

	// First pass: Count occurrences and store paths
	fileHashes.Range(func(path, value any) bool {
		hash := value.(models.FileHash).Hash
		hashCounts[hash]++
		hashPaths[hash] = append(hashPaths[hash], value.(models.FileHash))
		return true
	})

	// Second pass: Update duplicates in the map
	fileHashes.Range(func(path, value any) bool {
		hash := value.(models.FileHash).Hash
		if hashCounts[hash] > 1 {
			// Filter out the current file from duplicates
			duplicates := make([]models.FileHash, 0, len(hashPaths[hash])-1)
			for _, file := range hashPaths[hash] {
				if file.FilePath != path {
					duplicates = append(duplicates, file)
				}
			}
			// Update the file with duplicates
			fileHash := value.(models.FileHash)
			fileHash.DuplicatesFound = duplicates
			fileHashes.Store(path, fileHash)
		}
		return true
	})
}

func FindDuplicatesBetweenMaps(first *sync.Map, second *sync.Map) {
	// Iterate through each file hash in the first map
	first.Range(func(firstPath, firstVal any) bool {
		duplicates := []models.FileHash{}

		second.Range(func(secondPath, secValue any) bool {
			if secValue.(models.FileHash).Hash == firstVal.(models.FileHash).Hash {
				duplicates = append(duplicates, secValue.(models.FileHash))
			}
			return true
		})
		old, ok := first.Load(firstPath)
		var new models.FileHash
		if ok {
			new = old.(models.FileHash)
			new.DuplicatesFound = duplicates
		}
		first.Store(firstPath, new)

		return true
	})

	// for path, fileHash := range *first {
	// 	// Check for matches in the second map
	// 	duplicates := []models.FileHash{}

	// 	for _, otherFileHash := range *second {
	// 		if fileHash.Hash == otherFileHash.Hash {
	// 			// Mark the match as a duplicate
	// 			duplicates = append(duplicates, otherFileHash)

	// 		}
	// 	}
	// 	itemCopy := fileHash
	// 	itemCopy.DuplicatesFound = duplicates
	// 	// Update the map with the modified copy
	// 	(*first)[path] = itemCopy
	// }
}

func GetDuplicates(input *sync.Map) map[string]models.FileHash {

	seen := make(map[string]bool)
	result := make(map[string]models.FileHash)

	input.Range(func(key, value any) bool {
		value1, ok := input.Load(key)
		if ok {
			if !seen[value1.(models.FileHash).Hash] {
				seen[value1.(models.FileHash).Hash] = true
			} else {
				result[value1.(models.FileHash).Hash] = value1.(models.FileHash)
			}
		}

		return true
	})
	return result
}

func GetFlattened(input *map[string]models.FileHash) []models.ResultEntry {
	result := make([]models.ResultEntry, 0)
	separatorEntry := models.ResultEntry{Filename: "------", FullPath: "------", DuplicateFilename: "------", DuplicateFullPath: "------"}
	for _, val := range *input {
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
	}
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
