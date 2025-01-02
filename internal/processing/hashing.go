package processing

import (
	logger "DuDe/common"
	models "DuDe/models"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

func CreateHashes(sourceFiles *[]models.DuDeFile, maxWorkers int, progressCh chan int, memoryChan chan models.FileHash, memory *[]models.FileHash, enableMemory bool) error {

	var doneFiles int32 // Atomic counter for progress tracking

	var wg sync.WaitGroup
	mutex := sync.Mutex{}
	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size

	for i := range *sourceFiles {
		wg.Add(1)
		go func(index int) error {
			defer wg.Done()
			var hash string
			var err error
			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			if enableMemory {

				path := (*sourceFiles)[index].FullPath
				memoryOfFile := models.FindByPath(*memory, path)
				stats, _ := os.Stat(path)

				if memoryOfFile == nil ||
					memoryOfFile.FileSize != stats.Size() ||
					memoryOfFile.ModTime != stats.ModTime().Unix() {
					hash, err = calculateMD5Hash((*sourceFiles)[index])
					if err != nil {
						return err
					}

				} else {
					hash = memoryOfFile.Hash
				}
			}

			name := GetFileName((*sourceFiles)[index].FullPath)
			mutex.Lock()
			(*sourceFiles)[index].Hash = hash
			(*sourceFiles)[index].Filename = name
			mutex.Unlock()

			if enableMemory {

				fileStats, _ := os.Stat((*sourceFiles)[index].FullPath)
				newMem := models.FileHash{
					FilePath: (*sourceFiles)[index].FullPath,
					Hash:     (*sourceFiles)[index].Hash,
					FileSize: fileStats.Size(),
					ModTime:  fileStats.ModTime().Unix(),
				}

				*memory = append(*memory, newMem)
				memoryChan <- newMem
			}

			atomic.AddInt32(&doneFiles, 1)
			progressCh <- int(doneFiles)
			return nil
		}(i)
	}
	wg.Wait()

	close(sem)
	close(progressCh)
	// close(memoryChan)

	return nil
}

func calculateMD5Hash(file models.DuDeFile) (string, error) {
	hasherMD5 := md5.New()

	f, err := os.Open(file.FullPath)
	defer func() {
		err := f.Close()
		if err != nil {
			logger.PanicAndLog(err)
		}
	}()

	io.Copy(hasherMD5, f)
	return fmt.Sprintf("%x", hasherMD5.Sum(nil)), err
}

func FindDuplicates(input *[]models.DuDeFile) {
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
}

func GetDuplicates(input *[]models.DuDeFile) []models.DuDeFile {
	seen := make(map[string]bool)
	result := make([]models.DuDeFile, 0)
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

func GetFlattened(input *[]models.DuDeFile) []models.ResultEntry {
	result := make([]models.ResultEntry, 0)
	empty := models.ResultEntry{}
	for _, val := range *input {
		for _, dup := range val.DuplicatesFound {
			a := models.ResultEntry{
				Filename:          val.Filename,
				FullPath:          val.FullPath,
				DuplicateFilename: dup.Filename,
				DuplicateFullPath: dup.FullPath,
			}
			result = append(result, a)
		}
		result = append(result, empty)
	}
	return result
}
