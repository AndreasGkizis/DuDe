package processing

import (
	logger "DuDe/common"
	"DuDe/models"
	"sync"
)

var (
	records           = models.FileHashCollection{Hashes: make(map[string]models.FileHash)}
	mu                sync.Mutex
	runningSchedulers = make(chan struct{}, 1)
)

func StartMemoryUpdateBackground(path string, memoryChan <-chan models.FileHash) {
	var mutex sync.Mutex
	go func() {
		for newHash := range memoryChan {

			mutex.Lock()
			err := UpsertMemoryCSV(path, newHash)
			if err != nil {
				logger.PanicAndLog(err)
			}
			mutex.Unlock()
		}
	}()
}

func UpsertMemoryCSV(memoryPath string, info models.FileHash) error {
	mu.Lock()
	defer mu.Unlock()

	// Update or insert the record in memory
	records.Hashes[info.FilePath] = info

	// Optionally, write back to CSV periodically or based on some condition
	go scheduleWriter(memoryPath)

	return nil
}

func scheduleWriter(memoryPath string) {
	runningSchedulers <- struct{}{} // attempting to get semaphore

	logger.GetLogger().Debugf("got semaphore!")
	defer func() {
		<-runningSchedulers
		logger.GetLogger().Debugf("semaphore released!")
	}()

	// time.Sleep(3 * time.Second) // start writing in 3 second intervals

	mu.Lock()
	defer mu.Unlock()
	logger.GetLogger().Debugf("Writing to %s", memoryPath)

	WriteManyToMemoryCSV(memoryPath, records.ToSlice())
}
