package processing

import (
	log "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	models "DuDe/internal/models"
	"database/sql"
	"fmt"
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

func NewMemoryManager(args *models.ExecutionParams, bufferSize, senderCount int) *MemoryManager {
	var localdb *sql.DB
	var err error
	if args.UseCache {
		localdb, err = database.InitializeDatabase(args.CacheDir)
		if err != nil {
			log.ErrorWithFuncName(err.Error())
		}
	}

	return &MemoryManager{
		senderCount: int32(senderCount),
		Channel:     make(chan models.FileHash, bufferSize),
		repo:        *database.NewFileHashRepository(localdb),
		isActive:    args.UseCache}
}

// NewMemoryManagerWithDB creates a MemoryManager using an already-open *sql.DB.
// Intended for testing, where the caller controls the DB lifecycle.
func NewMemoryManagerWithDB(args *models.ExecutionParams, bufferSize, senderCount int, db *sql.DB) *MemoryManager {
	return &MemoryManager{
		senderCount: int32(senderCount),
		Channel:     make(chan models.FileHash, bufferSize),
		repo:        *database.NewFileHashRepository(db),
		isActive:    args.UseCache,
	}
}

func (mm *MemoryManager) Start() {
	if !mm.isActive {
		return
	}

	mm.wg.Add(1)
	mm.senderWg.Add(int(mm.senderCount))
	go mm.updateMemory()
}

func (mm *MemoryManager) LoadMemory() (map[string]models.FileHash, error) {
	if !mm.isActive {
		return make(map[string]models.FileHash), nil
	}

	records, err := mm.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load memory from cache: %w", err)
	}

	result := make(map[string]models.FileHash)
	for _, val := range records {
		result[val.FilePath] = MapToServiceDTO(val)
	}

	return result, nil
}

func (mm *MemoryManager) Wait() {

	if !mm.isActive {
		return
	}
	mm.wg.Wait()
	mm.senderWg.Wait()
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
			log.FatalWithFuncName(err.Error())
		}
	}

	log.DebugWithFuncName("finished")
}
