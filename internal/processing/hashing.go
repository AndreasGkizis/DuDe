package processing

import (
	"DuDe/common"
	logger "DuDe/common"
	visuals "DuDe/internal/visuals"
	models "DuDe/models"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func CreateHashes(sourceFiles *[]models.FileHash, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash) error {

	pt.AddTotal(int64(len(*sourceFiles)))
	mm.SenderStarted()

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size

	for i := range *sourceFiles {
		wg.Add(1)
		go func(index int) error {
			defer wg.Done()
			var hash string

			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			path := (*sourceFiles)[index].FilePath
			stats, _ := os.Stat(path)

			curSize := stats.Size()
			curModTime := stats.ModTime().Format(time.RFC3339) //TODO: encapsulate into model

			memoryOfFile, exists := (*memory)[path]
			needsRehasing := !exists || (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if needsRehasing {
				hash = calculateMD5Hash((*sourceFiles)[index])
			} else {
				hash = memoryOfFile.Hash
			}

			(*sourceFiles)[index].Hash = hash
			(*sourceFiles)[index].FileSize = curSize
			(*sourceFiles)[index].ModTime = curModTime
			(*sourceFiles)[index].FileName = filepath.Base(path)

			newMem := models.FileHash{
				FileName: filepath.Base(path),
				FilePath: path,
				Hash:     hash,
				FileSize: curSize,
				ModTime:  curModTime,
			}

			safeResend(mm.Channel, newMem, 2, 12*time.Microsecond)
			// mm.Channel <- newMem

			pt.Increment()
			return nil
		}(i)
	}

	wg.Wait()
	mm.SenderFinished()

	close(sem)

	return nil
}

func calculateMD5Hash(file models.FileHash) string {
	hasherMD5 := md5.New()

	f, err := os.Open(file.FilePath)
	if err != nil {
		logger.Logger.DPanic(err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logger.DPanic(err)
		}
	}()

	io.Copy(hasherMD5, f)
	return fmt.Sprintf("%x", hasherMD5.Sum(nil))
}

func FindDuplicates(inputs ...*[]models.FileHash) {

	if len(inputs) == 1 {
		input := inputs[0]

		for i := range *input {
			occurrenceCounter := 0
			for j := range *input {
				if (*input)[i].Hash == (*input)[j].Hash {
					if occurrenceCounter == 0 {
						occurrenceCounter++
					} else {
						(*input)[i].DuplicatesFound = append((*input)[i].DuplicatesFound, (*input)[j])
					}
				}
			}
		}
	} else if len(inputs) == 2 {

		first := inputs[0]
		second := inputs[1]

		for i := range *first {
			for j := range *second {
				if (*first)[i].Hash == (*second)[j].Hash {
					(*first)[i].DuplicatesFound = append((*first)[i].DuplicatesFound, (*second)[j])
				}
			}
		}
	}

}

func GetDuplicates(input *[]models.FileHash) []models.FileHash {
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

// safeResend attempts to send data to a buffered channel, retrying on failure.
func safeResend(ch chan<- models.FileHash, data models.FileHash, maxRetries int, retryDelay time.Duration) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case ch <- data:
			return // Successfully sent.
		default:
			if attempt < maxRetries {
				common.Logger.Warnf("Failed to send data (attempt %d/%d), retrying in %v: %v", attempt+1, maxRetries+1, retryDelay, data)
				time.Sleep(retryDelay)
			} else {
				common.Logger.Warnf("Failed to send data after %d attempts: %v", maxRetries+1, data)
				return // Give up after max retries.
			}
		}
	}
}

func sendWithRetry(ch chan models.FileHash, value models.FileHash, baseDelay time.Duration) error {
	for {
		select {
		case ch <- value:
			return nil // Success
		case <-time.After(time.Duration(baseDelay.Seconds())):
			fmt.Printf("Channel full, retrying send %s\n", value.Hash)
		}
	}
}
