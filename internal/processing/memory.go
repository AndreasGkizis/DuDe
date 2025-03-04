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

type MemoryTracker struct {
	repo database.FileHashRepository
	db   *sql.DB
	wg   sync.WaitGroup
}

func NewMemoryTracker(db *sql.DB) *MemoryTracker {
	return &MemoryTracker{
		repo: *database.NewFileHashRepository(db),
		db:   db}
}

func (mt *MemoryTracker) Start(ch <-chan models.FileHash) {
	mt.wg.Add(1)
	go mt.updateMemory(ch)
}

func (mt *MemoryTracker) updateMemory(memoryChan <-chan models.FileHash) {
	common.DebugWithFuncName(fmt.Sprintf("started at %s", time.Now()))

	for fh := range memoryChan {
		db_fh := MapToDomainDTO(fh)
		err := mt.repo.Upsert(&db_fh)
		common.PanicAndLog(err)
	}

	common.DebugWithFuncName(fmt.Sprintf("finished at %s", time.Now())) // NOTE: currently unreachable
}
