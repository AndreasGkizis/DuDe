package processing

import (
	logger "DuDe/common"
	"DuDe/internal/visuals"
	models "DuDe/models"
	"crypto/md5"
	"fmt"
	"io"
	"os"
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
			fmt.Println(index)
			var hash string
			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			path := (*sourceFiles)[index].FilePath
			stats, _ := os.Stat(path)

			memoryOfFile := models.FindByPath(memory, path)

			curSize := stats.Size()
			curModTime := stats.ModTime().Format(time.RFC3339) //TODO: ncap[sulate into model

			fileMemoryMissing := memoryOfFile == nil
			fileChanged := memoryOfFile != nil && (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if fileMemoryMissing {
				hash = calculateMD5Hash((*sourceFiles)[index])
			} else if fileChanged {
				hash = calculateMD5Hash((*sourceFiles)[index])
			} else {
				hash = memoryOfFile.Hash
			}

			newMem := models.FileHash{
				FileName: GetFileName(path),
				FilePath: path,
				Hash:     hash,
				FileSize: curSize,
				ModTime:  curModTime,
			}

			memoryChan <- newMem

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
