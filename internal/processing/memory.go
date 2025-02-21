package processing

import (
	logger "DuDe/common"
	"DuDe/models"
	"fmt"
	"sync"
	"time"
)

var (
	memory_buffer     []*models.FileHashBatch
	runningSchedulers = make(chan struct{}, 1) // change number to adjust concurrent shedulers
)

func StartMemoryUpdateBackgroundProcess(path string, memoryChan <-chan models.FileHash) {
	batchSize := uint(2) // for test reasons , should be larger
	fmt.Print("StartMemoryUpdateBackgroundProcess")

	ticker := time.NewTicker(5 * time.Second) // Write every 5 seconds
	defer ticker.Stop()

	var mutex sync.Mutex
	go func() {
		currentBatch := models.NewFileBatch(batchSize)
		for newHash := range memoryChan {
			mutex.Lock()
			// if full, unload and create a new one
			if len(currentBatch.Entries) >= int(currentBatch.BatchSize) {
				memory_buffer = append(memory_buffer, currentBatch)
				currentBatch = models.NewFileBatch(batchSize)
				mutex.Unlock() // Unlock after appending and before next iteration
				continue       // Important: Continue to avoid double unlock
			}

			currentBatch.Entries = append(currentBatch.Entries, newHash)
			mutex.Unlock()
		}
		// mutex.Lock()
		// // only adds to buffer if there is something in this batch
		// if len(currentBatch.Entries) > 0 {
		// 	memory_buffer = append(memory_buffer, currentBatch)
		// }
		// mutex.Unlock()

		scheduleWriter(path)
	}()
}

func scheduleWriter(memoryPath string) {
	fmt.Print("scheduleWriter")

	runningSchedulers <- struct{}{} // attempting to get semaphore
	var mutex sync.Mutex

	logger.GetLogger().Debugf("got semaphore!")
	defer func() {
		<-runningSchedulers
		logger.GetLogger().Debugf("semaphore released!")
	}()

	time.Sleep(3 * time.Second) // start writing in 3 second intervals

	mutex.Lock()
	defer mutex.Unlock()
	logger.GetLogger().Debugf("Writing to %s", memoryPath)
	for _, batch := range memory_buffer {
		err := WriteAllToMemoryCSV(memoryPath, batch.Entries)
		if err != nil {
			logger.PanicAndLog(err)
		}
		batch.Saved = true
	}
	// memory_buffer = nil // Clear the buffer after processing

}
