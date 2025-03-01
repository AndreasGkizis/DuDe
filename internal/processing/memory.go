package processing

import (
	common "DuDe/common"
	database "DuDe/internal/db"
	"DuDe/models"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

func UpdateMemory(db *sql.DB, memoryChan <-chan models.FileHash) {

	common.DebugWithFuncName("started")

	repo := database.FileHashRepository{Db: db}

	for fh := range memoryChan {
		db_fh := MapToDomainDTO(fh)
		err := repo.Upsert(&db_fh)
		common.PanicAndLog(err)
	}
}

func Remember(db *sql.DB, memoryChan <-chan models.FileHash) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Saving to memory...")

		done := make(chan struct{})

		go func() {
			UpdateMemory(db, memoryChan)
			close(done) // Signal completion
		}()

		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				fmt.Println("...Saved to memory!")
				return // Exit the goroutine
			case <-ticker.C:
				fmt.Println("Saving...Please wait")
			}
		}
	}()

	wg.Wait()
}

func LoadMemory(db *sql.DB) []models.FileHash {
	result := []models.FileHash{}
	repo := database.FileHashRepository{Db: db}

	records, err := repo.GetAll()

	common.PanicAndLog(err)
	for _, val := range records {
		result = append(result, MapToServiceDTO(val))
	}

	return result
}
