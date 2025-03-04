package processing

import (
	logger "DuDe/common"
	"DuDe/internal/visuals"
	models "DuDe/models"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func CreateHashes(sourceFiles *[]models.FileHash, maxWorkers int, pt *visuals.ProgressTracker, progressCh chan int, memoryChan chan models.FileHash, memory *[]models.FileHash) error {

	pt.AddTotal(int64(len(*sourceFiles)))

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size

	for i := range *sourceFiles {
		wg.Add(1)
		go func(index int) error {
			var hash string

			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			path := (*sourceFiles)[index].FilePath
			stats, _ := os.Stat(path)

			memoryOfFile := models.FindByPath(memory, path)

			curSize := stats.Size()
			curModTime := stats.ModTime().Format(time.RFC3339) //TODO: encapsulate into model

			fileMemoryMissing := memoryOfFile == nil
			fileChanged := memoryOfFile != nil && (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if fileMemoryMissing || fileChanged {
				hash = calculateMD5Hash((*sourceFiles)[index])
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

			sendWithRetry(memoryChan, newMem, 1, 150*time.Millisecond)

			progressCh <- 1
			pt.Increment()
			wg.Done()
			return nil
		}(i)
	}
	wg.Wait()

	close(sem)

	return nil
}

func calculateMD5Hash(file models.FileHash) string {
	hasherMD5 := md5.New()

	f, err := os.Open(file.FilePath)
	if err != nil {
		logger.PanicAndLog(err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			logger.PanicAndLog(err)
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
			// occurrenceCounter := 0
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

func sendWithRetry(ch chan models.FileHash, value models.FileHash, maxRetries int, baseDelay time.Duration) error {
	// if maxRetries < 0 {
	for {
		select {
		case ch <- value:
			// fmt.Printf("Sent! %s \n", value.Hash)
			return nil // Success
		case <-time.After(time.Duration(rand.Int63n(int64(baseDelay / 2)))):
			fmt.Printf("Channel full, retrying send %s\n", value.Hash)
		}
	}
	// } else {
	// 	for attempt := 0; attempt < maxRetries; attempt++ {
	// 		select {
	// 		case ch <- value:
	// 			fmt.Printf("Sent! %s \n", value.Hash)
	// 			return nil // Success
	// 		case <-time.After(baseDelay*time.Duration(1<<uint(attempt)) + time.Duration(rand.Int63n(int64(baseDelay/2)))):
	// 			if attempt == maxRetries-1 {
	// 				return fmt.Errorf("failed to send %d after %d retries", value, maxRetries)
	// 			}
	// 			fmt.Printf("Channel full, retrying send %d (attempt %d)\n", value, attempt+1)
	// 		}
	// 	}
	// }
	// return nil
}
