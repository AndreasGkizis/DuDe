package processing

import (
	logger "DuDe/common"
	"DuDe/models"
	"sync"
)

func StartMemoryUpdateBackground(path string, memoryChan <-chan models.FileHash) {
	var mutex sync.Mutex
	go func() {
		for newHash := range memoryChan {

			mutex.Lock()
			err := AddSingleToMemoryCSV(path, newHash)
			if err != nil {
				logger.PanicAndLog(err)
			}
			mutex.Unlock()
		}
	}()
}
