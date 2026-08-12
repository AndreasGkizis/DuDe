package processing

import (
	log "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	models "DuDe/internal/models"
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type MemoryManager struct {
	Channel     chan models.FileHash
	db          *sql.DB
	repo        *database.FileHashRepository
	wg          sync.WaitGroup
	senderWg    sync.WaitGroup
	senderCount int32
	isActive    atomic.Bool
	started     atomic.Bool
	queued      int64
	completed   int64
	processed   int64
	errorMu     sync.Mutex
	cacheErr    error
	closeOnce   sync.Once
	channelOnce sync.Once
}

func NewMemoryManager(args *models.ExecutionParams, bufferSize, senderCount int) *MemoryManager {
	manager := &MemoryManager{
		senderCount: int32(senderCount),
		Channel:     make(chan models.FileHash, bufferSize),
	}
	if !args.UseCache {
		return manager
	}

	localDB, err := database.InitializeDatabase(args.CacheDir)
	if err != nil {
		manager.recordCacheFailure(fmt.Errorf("initialize cache: %w", err))
		return manager
	}

	manager.db = localDB
	manager.repo = database.NewFileHashRepository(localDB)
	manager.isActive.Store(true)
	return manager
}

func (mm *MemoryManager) CacheError() error {
	mm.errorMu.Lock()
	defer mm.errorMu.Unlock()
	return mm.cacheErr
}

func (mm *MemoryManager) Start() {
	if !mm.isActive.Load() || mm.started.Load() {
		return
	}

	mm.wg.Add(1)
	mm.senderWg.Add(int(mm.senderCount))
	mm.started.Store(true)
	go mm.updateMemory()
}

func (mm *MemoryManager) LoadMemory() map[string]models.FileHash {
	result := make(map[string]models.FileHash)

	if !mm.isActive.Load() {
		return result
	}

	records, err := mm.repo.GetAll()
	if err != nil {
		mm.recordCacheFailure(fmt.Errorf("load cache: %w", err))
		mm.closeDatabase()
		return result
	}

	for _, val := range records {
		result[val.FilePath] = MapToServiceDTO(val)
	}

	return result
}

func (mm *MemoryManager) Wait() {
	if !mm.started.Load() {
		return
	}
	mm.wg.Wait()
	mm.senderWg.Wait()
}

func (mm *MemoryManager) SenderFinished() {
	if !mm.started.Load() {
		return
	}

	for {
		remaining := atomic.LoadInt32(&mm.senderCount)
		if remaining <= 0 {
			return
		}
		if !atomic.CompareAndSwapInt32(&mm.senderCount, remaining, remaining-1) {
			continue
		}

		mm.senderWg.Done()
		if remaining == 1 {
			mm.channelOnce.Do(func() {
				close(mm.Channel)
			})
		}
		return
	}
}

func (mm *MemoryManager) CloseAndWait() {
	if !mm.started.Load() {
		return
	}

	for atomic.LoadInt32(&mm.senderCount) > 0 {
		mm.SenderFinished()
	}
	mm.Wait()
}

func (mm *MemoryManager) Push(fh models.FileHash) {
	if !mm.isActive.Load() {
		return
	}
	atomic.AddInt64(&mm.queued, 1)
	mm.Channel <- fh
}

func (mm *MemoryManager) WaitForCache(ctx context.Context, progress func(completed, total int64)) bool {
	if !mm.started.Load() {
		return false
	}

	total := atomic.LoadInt64(&mm.queued)
	processed := atomic.LoadInt64(&mm.processed)
	if total == 0 || processed >= total {
		mm.Wait()
		return false
	}

	progress(processed, total)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for processed < total {
		select {
		case <-ctx.Done():
			mm.Wait()
			return true
		case <-ticker.C:
			processed = atomic.LoadInt64(&mm.processed)
			progress(processed, total)
		}
	}

	mm.Wait()
	return true
}

func (mm *MemoryManager) CacheProgress() (completed, total int64, enabled bool) {
	return atomic.LoadInt64(&mm.completed), atomic.LoadInt64(&mm.queued), mm.isActive.Load()
}

func (mm *MemoryManager) updateMemory() {
	log.DebugWithFuncName("started")
	defer mm.wg.Done()
	defer mm.closeDatabase()

	for fh := range mm.Channel {
		if mm.isActive.Load() {
			databaseFileHash := MapToDomainDTO(fh)
			if err := mm.repo.Upsert(&databaseFileHash); err != nil {
				mm.recordCacheFailure(fmt.Errorf("write cache: %w", err))
			} else {
				atomic.AddInt64(&mm.completed, 1)
			}
		}
		atomic.AddInt64(&mm.processed, 1)
	}

	log.DebugWithFuncName("finished")
}

func (mm *MemoryManager) recordCacheFailure(err error) {
	mm.errorMu.Lock()
	if mm.cacheErr == nil {
		mm.cacheErr = err
		log.WarnWithFuncName(fmt.Sprintf("Cache unavailable; continuing without cache: %v", err))
	}
	mm.errorMu.Unlock()
	mm.isActive.Store(false)
}

func (mm *MemoryManager) closeDatabase() {
	mm.closeOnce.Do(func() {
		if mm.db == nil {
			return
		}
		if err := mm.db.Close(); err != nil {
			mm.recordCacheFailure(fmt.Errorf("close cache: %w", err))
		}
	})
}
