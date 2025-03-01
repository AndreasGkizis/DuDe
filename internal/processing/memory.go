package processing

import (
	common "DuDe/common"
	database "DuDe/internal/db"
	"DuDe/models"
	"sync"

	"gorm.io/gorm"
)

func UpdateMemory(db *gorm.DB, memoryChan <-chan models.FileHash) {

	common.DebugWithFuncName("started")

	repo := database.FileHashRepository{Db: db}

	for fh := range memoryChan {
		db_obj := MapToDomainDTO(fh)
		err := repo.Upsert(&db_obj)
		common.PanicAndLog(err)
	}
}

func Remember(db *gorm.DB, memoryChan <-chan models.FileHash) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		UpdateMemory(db, memoryChan)
	}()
	wg.Wait()
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
