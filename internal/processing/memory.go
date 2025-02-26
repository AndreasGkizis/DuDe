package processing

import (
	common "DuDe/common"
	database "DuDe/internal/db"
	"DuDe/models"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

var (
	memory_buffer     []*models.FileHashBatch
	runningSchedulers = make(chan struct{}, 1) // change number to adjust concurrent schedulers
)

func StartMemoryUpdateBackgroundProcess(path string, memoryChan <-chan models.FileHash) {
	batchSize := uint(2) // for test reasons , should be larger
	fmt.Print("StartMemoryUpdateBackgroundProcess")

	ticker := time.NewTicker(5 * time.Millisecond) // Write every 5 seconds
	defer ticker.Stop()

	var mutex sync.Mutex
	go func() {
		currentBatch := models.NewFileBatch(batchSize)

		for {
			select {
			case newHash := <-memoryChan:
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
			case t := <-ticker.C:
				mutex.Lock()
				common.Logger.Log(zapcore.DebugLevel, t)
				// only adds to buffer if there is something in this batch
				if len(currentBatch.Entries) > 0 { // Write even if batch is not full
					memory_buffer = append(memory_buffer, currentBatch)
					currentBatch = models.NewFileBatch(batchSize) // Start a new batch
				}
				mutex.Unlock()
				scheduleWriter(path)
			default:
				mutex.Lock()
				// only adds to buffer if there is something in this batch
				if len(currentBatch.Entries) > 0 { // Write even if batch is not full
					memory_buffer = append(memory_buffer, currentBatch)
					currentBatch = models.NewFileBatch(batchSize) // Start a new batch
				}
				mutex.Unlock()
				go scheduleWriter(path)
			}
		}
	}()
}

func UpdateMemory(db *gorm.DB, memoryChan <-chan models.FileHash) {
	common.Logger.Info("started memory.UpdateMemory()")
	repo := database.FileHashRepository{Db: db}

	for fh := range memoryChan {
		common.Logger.Debugf("Go one from %s. path: %s, hash: %s", memoryChan, fh.FilePath, fh.Hash)
		db_obj := MapToDomainDTO(fh)
		err := repo.Upsert(&db_obj)
		common.PanicAndLog(err)
	}
}

func Updatewitwg(db *gorm.DB, memoryChan <-chan models.FileHash) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		UpdateMemory(db, memoryChan)
	}()
	wg.Wait()
}
func scheduleWriter(memoryPath string) {
	fmt.Print("scheduleWriter")

	runningSchedulers <- struct{}{} // attempting to get semaphore
	var mutex sync.Mutex

	common.Logger.Debugf("got semaphore!")
	defer func() {
		<-runningSchedulers
		common.Logger.Debugf("semaphore released!")
	}()

	// time.Sleep(3 * time.Second) // start writing in 3 second intervals

	mutex.Lock()
	defer mutex.Unlock()
	common.Logger.Debugf("Writing to %s", memoryPath)
	for _, batch := range memory_buffer {
		err := WriteAllToMemoryCSV(memoryPath, batch.Entries)
		if err != nil {
			common.PanicAndLog(err)
		}
		batch.Saved = true
	}
	// memory_buffer = nil // Clear the buffer after processing

}

func LoadMemory(db *gorm.DB) []models.FileHash {
	result := []models.FileHash{}
	repo := database.FileHashRepository{Db: db}

	records, err := repo.GetAll()
	common.PanicAndLog(err)
	for _, val := range records {
		result = append(result, MapToServiceDTO(val))
	}

	return result
}
