package processing

import (
	"DuDe/internal/common"
	log "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	models "DuDe/internal/models"
	"database/sql"
	"sync"
	"sync/atomic"
)

type MemoryManager struct {
	Channel     chan models.FileHash
	repo        database.FileHashRepository
	wg          sync.WaitGroup
	senderWg    sync.WaitGroup
	senderCount int32
	isActive    bool
}

func NewMemoryManager(args *models.ExecutionParams, db *sql.DB, bufferSize, senderCount int) *MemoryManager {
	return &MemoryManager{
		senderCount: int32(senderCount),
		Channel:     make(chan models.FileHash, bufferSize),
		repo:        *database.NewFileHashRepository(db),
		isActive:    args.UseCache}
}

func (mm *MemoryManager) Start() {
	if !mm.isActive {
		return
	}

	mm.wg.Add(1)
	mm.senderWg.Add(int(mm.senderCount))
	go mm.updateMemory()
}

func (mm *MemoryManager) LoadMemory() map[string]models.FileHash {
	result := make(map[string]models.FileHash)

	if !mm.isActive { // return empty memory
		return make(map[string]models.FileHash)
	}

	records := common.Must(mm.repo.GetAll())

	for _, val := range records {
		result[val.FilePath] = MapToServiceDTO(val)
	}

	return result
}

func (mm *MemoryManager) Wait() {

	if !mm.isActive {
		return
	}
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
	if !mm.isActive {
		return
	}

	if atomic.AddInt32(&mm.senderCount, -1) == 0 {
		close(mm.Channel)
	}
	mm.senderWg.Done()
}

func (mm *MemoryManager) Push(fh models.FileHash) {
	if !mm.isActive {
		return
	}
	mm.Channel <- fh
}

func (mm *MemoryManager) updateMemory() {
	log.DebugWithFuncName("started")
	defer mm.wg.Done()

	for fh := range mm.Channel {
		db_fh := MapToDomainDTO(fh)
		err := mm.repo.Upsert(&db_fh)
		if err != nil {
			log.Logger.Fatalf(err.Error())
		}
	}

	log.DebugWithFuncName("finished")
}
