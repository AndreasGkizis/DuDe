package processing

import (
	"DuDe/internal/common"
	logger "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	models "DuDe/internal/models"
	"database/sql"
	"sync"
	"sync/atomic"
)

type MemoryManager struct {
	Channel     chan models.FileHash
	senderCount int32
	repo        database.FileHashRepository
	wg          sync.WaitGroup
	senderWg    sync.WaitGroup
}

func NewMemoryManager(db *sql.DB, bufferSize int) *MemoryManager {
	return &MemoryManager{
		Channel: make(chan models.FileHash, bufferSize),
		repo:    *database.NewFileHashRepository(db)}
}

func (mm *MemoryManager) Start() {
	mm.wg.Add(1)
	go mm.updateMemory()
}

func (mm *MemoryManager) LoadMemory() map[string]models.FileHash {
	result := make(map[string]models.FileHash)

	records := common.Must(mm.repo.GetAll())

	for _, val := range records {
		result[val.FilePath] = MapToServiceDTO(val)
	}

	return result
}

func (mm *MemoryManager) Wait() {
	mm.wg.Wait()
	mm.senderWg.Wait()
}

func (mm *MemoryManager) SenderStarted() {
	atomic.AddInt32(&mm.senderCount, 1)
	mm.senderWg.Add(1)
}

func (mm *MemoryManager) TotalSenders(total int32) {
	atomic.AddInt32(&mm.senderCount, total)
	mm.senderWg.Add(int(total))
}

func (mm *MemoryManager) SenderFinished() {
	if atomic.AddInt32(&mm.senderCount, -1) == 0 {
		close(mm.Channel)
	}
	mm.senderWg.Done()
}

func (mm *MemoryManager) updateMemory() {
	logger.InfoWithFuncName("started")
	defer mm.wg.Done()

	for fh := range mm.Channel {
		db_fh := MapToDomainDTO(fh)
		err := mm.repo.Upsert(&db_fh)
		if err != nil {
			logger.Logger.Fatalf(err.Error())
		}
	}

	logger.InfoWithFuncName("finished")
}
