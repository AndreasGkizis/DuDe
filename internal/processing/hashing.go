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

func CreateHashes(sourceFiles *[]models.FileHash, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash, failedCount *int) error {
	groupID := rand.Uint32()
	logger.InfoWithFuncName(fmt.Sprintf("Group %d started hashing %d files with %d workers", groupID, int64(len(*sourceFiles)), maxWorkers))
	pt.AddTotal(int64(len(*sourceFiles)))

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
			curModTime := stats.ModTime().Format(time.RFC3339)

			memoryOfFile, exists := (*memory)[path]
			needsRehasing := !exists || (memoryOfFile.FileSize != curSize || memoryOfFile.ModTime != curModTime)

			if needsRehasing {
				var err error
				hash, err = calculateMD5Hash((*sourceFiles)[index])
				if err != nil {
					return nil
				}
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

	return nil
}

func EnsureDuplicates(input []models.FileHash, pt *visuals.ProgressTracker, maxWorkers int) ([]models.FileHash, error) {
	num := 0
	for _, v := range input {
		num += len(v.DuplicatesFound)
	}
	if num == 0 {
		logger.InfoWithFuncName("No duplicates to ensure")
		return input, nil
	}
	pt.AddTotal(int64(num))

	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	sem := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	var once sync.Once

	for valueidx := range input {
		wg.Add(1)
		go func(valueIndex int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			filehash := &input[valueIndex]

			if len(filehash.DuplicatesFound) == 0 {
				return
			}
			mainfile, err := os.Open(filehash.FilePath)
			if err != nil {
				once.Do(func() { errChan <- fmt.Errorf("error opening main file: %w", err) })
				return
			}

			defer mainfile.Close()

			for dupindex := 0; dupindex < len(filehash.DuplicatesFound); {
				dup := filehash.DuplicatesFound[dupindex]

				eq, err := filesEqual(mainfile, dup.FilePath)

				if err != nil {
					once.Do(func() { errChan <- err })
					return
				}

				if !eq {
					mu.Lock()
					filehash.DuplicatesFound = append(filehash.DuplicatesFound[:dupindex], filehash.DuplicatesFound[dupindex+1:]...)
					if len(filehash.DuplicatesFound) == 0 {
						input = append(input[:valueidx], input[valueidx+1:]...)
					}
					mu.Unlock()
				} else {
					dupindex++
				}
				// reset readers
				_, _ = mainfile.Seek(0, io.SeekStart)
				pt.Increment()
			}
		}(valueidx)
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
			return "", err // TODO: this swallows erros at the moment, need to think a way to handle
		}
		logger.Logger.DPanic(err)
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
