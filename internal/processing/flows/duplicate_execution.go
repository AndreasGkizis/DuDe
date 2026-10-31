//go:build !debug_progress

package flows

import (
	"DuDe/internal/common"
	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"DuDe/internal/processing/services"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrNoDirectories = errors.New("at least one directory must be provided")

// ReadableDirectoryValidator checks that a flow input directory can be read.
type ReadableDirectoryValidator interface {
	ReadableDir(path string) error
}

// DuplicateFlow is the duplicate-finding execution flow.
type DuplicateFlow struct {
	paths ReadableDirectoryValidator
}

// NewDuplicateFlow creates the duplicate-finding execution flow.
func NewDuplicateFlow(paths ReadableDirectoryValidator) DuplicateFlow {
	return DuplicateFlow{paths: paths}
}

func (DuplicateFlow) Mode() models.ExecutionMode {
	return models.ExecutionModeDuplicates
}

func (flow DuplicateFlow) Validate(args *models.ExecutionParams) error {
	if len(args.Directories) == 0 {
		return fmt.Errorf("Directories: %w", ErrNoDirectories)
	}

	for i, dir := range args.Directories {
		if err := flow.paths.ReadableDir(dir); err != nil {
			return fmt.Errorf("Directories[%d] (%q): %w", i, dir, err)
		}
	}
	return nil
}

// Run finds duplicate file groups and publishes the completed duplicate result.
func (flow DuplicateFlow) Run(ctx context.Context, args models.ExecutionParams, reporter reporting.Reporter, publisher ResultPublisher) error {
	log.Initialize(args.DebugMode)
	timer := time.Now()
	log.LogModelArgs(args)

	errorLogger := newExecutionErrorLogger(100)
	defer errorLogger.CloseAndWait()

	failedCounter := 0
	memory := services.NewMemoryManager(&args, args.BufferSizeBytes, 1)
	hashMemory := memory.LoadMemory()
	cacheWarningReported := false
	if cacheErr := memory.CacheError(); cacheErr != nil {
		reporter.LogDetailedStatus(ctx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
		cacheWarningReported = true
	}
	memory.Start()
	defer memory.CloseAndWait()

	files, err := services.NewFileDiscovery().Discover(ctx, args.Directories, reporter)
	if err != nil {
		return err
	}

	fileCount := common.LenSyncMap(files)
	reporter.LogProgress(ctx, "Reading", 100)
	reporter.LogFilesCount(ctx, int64(fileCount), int64(fileCount))
	if fileCount == 0 {
		reporter.LogProgress(ctx, "Error", 0)
		reporter.LogDetailedStatus(ctx, "No files found in directory/directories! Check your paths again")
		publisher.PublishDuplicateResult(models.DuplicateResult{})
		return nil
	}

	reporter.LogProgress(ctx, "Filtering", 0)
	reporter.LogFilesCount(ctx, 0, int64(fileCount))
	candidateCount, skippedCount := services.FilterHashCandidatesBySize(files)
	reporter.LogProgress(ctx, "Filtering", 100)
	reporter.LogFilesCount(ctx, int64(fileCount), int64(fileCount))
	log.InfoWithFuncName(fmt.Sprintf("Skipped %d files with unique sizes; %d files remain as hash candidates", skippedCount, candidateCount))
	if candidateCount == 0 {
		reporter.LogDetailedStatus(ctx, "No possible duplicates found: every file has a unique size")
		reporter.LogFilesCount(ctx, int64(fileCount), int64(fileCount))
		reporter.LogProgress(ctx, "Done", 100)
		reporter.FinishExecution(ctx)
		publisher.PublishDuplicateResult(models.DuplicateResult{})
		return nil
	}

	hashing := visuals.NewProgressTracker(ctx, reporter, "Hashing")
	hashing.Start()
	if err := services.CreateHashes(ctx, files, args.WorkerCount, hashing, memory, &hashMemory, &failedCounter, errorLogger.channel); err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}
	hashing.Wait()

	memory.WaitForCache(ctx, func(completed, total int64) {
		percentage := float64(completed) / float64(total) * 100
		reporter.LogProgress(ctx, "Caching", percentage)
		reporter.LogFilesCount(ctx, completed, total)
	})
	if cacheErr := memory.CacheError(); cacheErr != nil && !cacheWarningReported {
		reporter.LogDetailedStatus(ctx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
	}

	finding := visuals.NewProgressTracker(ctx, reporter, "Finding")
	finding.Start()
	services.FindDuplicatesInMap(ctx, files, finding)
	finding.Wait()

	length := common.LenSyncMap(files)
	log.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 && args.ParanoidMode {
		comparing := visuals.NewProgressTracker(ctx, reporter, "Comparing")
		comparing.Start()
		compareErr := services.EnsureDuplicates(ctx, files, comparing, args.WorkerCount)
		comparing.Wait()
		if compareErr != nil {
			return fmt.Errorf("verify duplicates: %w", compareErr)
		}
	}

	groupsToCollect := common.LenSyncMap(files)
	reporter.LogProgress(ctx, "Collecting", 0)
	reporter.LogFilesCount(ctx, 0, int64(groupsToCollect))
	var groups []models.FileHash
	collectedGroups := 0
	files.Range(func(_, value any) bool {
		if fileHash, ok := value.(models.FileHash); ok && len(fileHash.DuplicatesFound) > 0 {
			groups = append(groups, fileHash)
		}
		collectedGroups++
		return true
	})
	reporter.LogProgress(ctx, "Collecting", 100)
	reporter.LogFilesCount(ctx, int64(collectedGroups), int64(groupsToCollect))

	length = common.LenSyncMap(files)
	if length != 0 {
		savingStarted := time.Now()
		if err := services.SaveResultsAsCSV(files, args.ResultsDir); err != nil {
			log.FatalWithFuncName(fmt.Sprintf("Error saving result: %v", err))
			return err
		}
		log.InfoWithFuncName(fmt.Sprintf("Took: %s to look through bytes", time.Since(savingStarted)))
	} else {
		log.InfoWithFuncName("No duplicates were found")
	}

	log.InfoWithFuncName(fmt.Sprintf("Took: %s for buffer size %d", time.Since(timer), args.BufferSizeBytes))
	log.InfoWithFuncName(fmt.Sprintf("Failed %d times to send to memoryChan", failedCounter))
	reporter.LogFilesCount(ctx, int64(fileCount), int64(fileCount))
	reporter.LogProgress(ctx, "Done", 100)
	reporter.FinishExecution(ctx)
	publisher.PublishDuplicateResult(models.DuplicateResult{Groups: groups})
	return nil
}

type executionErrorLogger struct {
	channel chan error
	wg      sync.WaitGroup
	once    sync.Once
}

func newExecutionErrorLogger(bufferSize int) *executionErrorLogger {
	logger := &executionErrorLogger{channel: make(chan error, bufferSize)}
	logger.wg.Add(1)
	go func() {
		defer logger.wg.Done()
		for err := range logger.channel {
			log.WarnWithFuncName(err.Error())
		}
	}()
	return logger
}

func (logger *executionErrorLogger) CloseAndWait() {
	logger.once.Do(func() { close(logger.channel) })
	logger.wg.Wait()
}
