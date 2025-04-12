package processing

import (
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

func CreateHashes(sourceFiles *map[string]models.FileHash, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash, failedCount *int, errChan chan error) error {
	groupID := rand.Uint32()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d started hashing %d files with %d workers", groupID, int64(len(*sourceFiles)), maxWorkers))
	pt.AddTotal(int64(len(*sourceFiles)))

	var wg sync.WaitGroup
	var mu sync.Mutex
	fmt.Println()
	fmt.Print(len(*sourceFiles))
	fmt.Println()
	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size
	// var globalError error                  // To store the first error that occurs
	for i, val := range *sourceFiles {
		wg.Add(1)
		go func(path string) error {
			defer wg.Done()
			var hash string
			var err error

			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			// path := (*sourceFiles)[index].FilePath
			stats, err := os.Stat(val.FilePath)
			if err != nil {
				errChan <- err
				mu.Lock()
				delete(*sourceFiles, val.FilePath)
				mu.Unlock()
				pt.DecrementFromTotal() // remove for progress bar
				return nil              // stop this iteration

			}

			curSize := stats.Size()
			curModTime := stats.ModTime().Format(time.RFC3339)

			memoryOfFile, exists := (*memory)[path]
			needsReHashing := !exists || (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if needsReHashing {
				hash, err = calculateMD5Hash(val)

				if err != nil {
					mu.Lock()
					delete(*sourceFiles, val.FilePath)

					mu.Unlock()
					pt.DecrementFromTotal() // remove for progress bar
					errChan <- err
					// if errors.Is(err, os.ErrPermission) {

					// mu.Lock()
					// // if globalError == nil { // Only store the first error
					// // 	globalError = err
					// // }
					// mu.Unlock()

					// }
					return nil // stop this iteration
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

			mu.Lock()
			(*sourceFiles)[path] = newMem
			mu.Unlock()

			// safeResend(mm.Channel, newMem, 500*time.Microsecond)
			sendWithRetry(mm.Channel, newMem, 500*time.Millisecond, 5*time.Second, failedCount)

			pt.Increment()
			return nil
		}(i)
	}

	wg.Wait()
	mm.SenderFinished()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d finished hashing %d files with %d workers", groupID, int64(len(*sourceFiles)), maxWorkers))

	close(sem)

	return nil // Return the first error that occurred, or nil if none
}

func EnsureDuplicates(input []models.FileHash, pt *visuals.ProgressTracker, maxWorkers int) ([]models.FileHash, error) {
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

	for valueIndx := range input {
		wg.Add(1)
		go func(valueIndex int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			fileHash := &input[valueIndex]

			if len(fileHash.DuplicatesFound) == 0 {
				return
			}
			mainFile, err := os.Open(fileHash.FilePath)
			if err != nil {
				once.Do(func() { errChan <- fmt.Errorf("error opening main file: %w", err) })
				return
			}

			defer mainFile.Close()

			for dupIndex := 0; dupIndex < len(fileHash.DuplicatesFound); {
				dup := fileHash.DuplicatesFound[dupIndex]

				eq, err := filesEqual(mainFile, dup.FilePath)

				if err != nil {
					once.Do(func() { errChan <- err })
					return
				}

				if !eq {
					mu.Lock()
					fileHash.DuplicatesFound = append(fileHash.DuplicatesFound[:dupIndex], fileHash.DuplicatesFound[dupIndex+1:]...)
					if len(fileHash.DuplicatesFound) == 0 {
						input = append(input[:valueIndx], input[valueIndx+1:]...)
					}
					mu.Unlock()
				} else {
					dupIndex++
				}
				// reset readers
				_, _ = mainFile.Seek(0, io.SeekStart)
				pt.Increment()
			}
		}(valueIndx)
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

func FindDuplicatesInMap(fileHashes *map[string]models.FileHash) {
	for i, iitem := range *fileHashes {
		duplicates := []models.FileHash{}
		occurrenceCounter := 0
		for _, jitem := range *fileHashes {
			if iitem.Hash == jitem.Hash {
				if occurrenceCounter == 0 {
					occurrenceCounter++
				} else {
					duplicates = append(duplicates, jitem)
				}
			}
		}
		// Create a copy of iitem to modify
		itemCopy := iitem
		itemCopy.DuplicatesFound = duplicates
		// Update the map with the modified copy
		(*fileHashes)[i] = itemCopy
	}
}

func FindDuplicatesBetweenMaps(first *map[string]models.FileHash, second *map[string]models.FileHash) {
	// Iterate through each file hash in the first map
	for path, fileHash := range *first {
		// Check for matches in the second map
		duplicates := []models.FileHash{}

		for _, otherFileHash := range *second {
			if fileHash.Hash == otherFileHash.Hash {
				// Mark the match as a duplicate
				duplicates = append(duplicates, otherFileHash)

			}
		}
		itemCopy := fileHash
		itemCopy.DuplicatesFound = duplicates
		// Update the map with the modified copy
		(*first)[path] = itemCopy
	}
}

func GetDuplicates(input *map[string]models.FileHash) []models.FileHash {
	seen := make(map[string]bool)
	result := make([]models.FileHash, 0)

	for _, val := range *input {
		if len(val.DuplicatesFound) > 0 {
			if !seen[val.Hash] {
				seen[val.Hash] = true
				result = append(result, val)
			}
		}
	}
	return result
}

func GetFlattened(input *[]models.FileHash) []models.ResultEntry {
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
