package processing

import (
	common "DuDe/common"
	database "DuDe/internal/db"
	models "DuDe/models"
	"database/sql"
	"sync"
	"sync/atomic"
)

type MemoryManager struct {
	Channel     chan models.FileHash
	senderCount int32
	repo        database.FileHashRepository
	db          *sql.DB
	wg          sync.WaitGroup
	senderwg    sync.WaitGroup
}

func NewMemoryManager(db *sql.DB, bufferSize int) *MemoryManager {
	return &MemoryManager{
		Channel: make(chan models.FileHash, bufferSize),
		repo:    *database.NewFileHashRepository(db),
		db:      db}
}

func (mt *MemoryManager) Start() {
	mt.wg.Add(1)
	go mt.updateMemory()
}

func (mt *MemoryManager) LoadMemory() []models.FileHash {
	result := []models.FileHash{}

	records, err := mt.repo.GetAll()

	if err != nil {
		common.Logger.DPanic(err)
	}

	for _, val := range records {
		result = append(result, MapToServiceDTO(val))
	}

	return result
}

func (mt *MemoryManager) Wait() {
	mt.wg.Wait()
	mt.senderwg.Wait()
}

func (mt *MemoryManager) SenderStarted() {
	atomic.AddInt32(&mt.senderCount, 1)
	mt.senderwg.Add(1)
}

func (mt *MemoryManager) SenderFinished() {
	if atomic.AddInt32(&mt.senderCount, -1) == 0 {
		close(mt.Channel)
	}
	mt.senderwg.Done()
}

func (mt *MemoryManager) updateMemory() {
	common.DebugWithFuncName("started")
	defer mt.wg.Done()

	for fh := range mt.Channel {
		db_fh := MapToDomainDTO(fh)
		err := mt.repo.Upsert(&db_fh)
		if err != nil {
			common.Logger.Fatalf(err.Error())
		}
	}

	common.DebugWithFuncName("finished") // NOTE: currently unreachable
}
