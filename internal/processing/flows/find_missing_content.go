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

var (
	ErrSourceDirectoryRequired = errors.New("source directory must be provided")
	ErrTargetDirectoryRequired = errors.New("target directory must be provided")
)

// FindMissingContentFlow owns Find Missing Content validation and execution.
type FindMissingContentFlow struct {
	paths ReadableDirectoryValidator
}

func NewFindMissingContentFlow(paths ReadableDirectoryValidator) FindMissingContentFlow {
	return FindMissingContentFlow{paths: paths}
}

func (FindMissingContentFlow) Mode() models.ExecutionMode {
	return models.ExecutionModeFindMissingContent
}

func (flow FindMissingContentFlow) Validate(args *models.ExecutionParams) error {
	if args.FindMissingSourceDir == "" {
		return fmt.Errorf("FindMissingSourceDir: %w", ErrSourceDirectoryRequired)
	}
	if args.FindMissingTargetDir == "" {
		return fmt.Errorf("FindMissingTargetDir: %w", ErrTargetDirectoryRequired)
	}

	if err := flow.paths.ReadableDir(args.FindMissingSourceDir); err != nil {
		return fmt.Errorf("FindMissingSourceDir (%q): %w", args.FindMissingSourceDir, err)
	}
	if err := flow.paths.ReadableDir(args.FindMissingTargetDir); err != nil {
		return fmt.Errorf("FindMissingTargetDir (%q): %w", args.FindMissingTargetDir, err)
	}

	return nil
}

// Run discovers, hashes, and matches source files against target files. Files
// skipped by discovery are reported separately and do not abort the run.
func (flow FindMissingContentFlow) Run(ctx context.Context, args models.ExecutionParams, reporter reporting.Reporter, publisher ResultPublisher) error {
	//start-up
	log.Initialize(args.DebugMode)
	timer := time.Now()
	log.LogModelArgs(args)

	memory := services.NewMemoryManager(&args, args.BufferSizeBytes, 2)
	defer memory.CloseAndWait()

	hashMemory := memory.LoadMemory()
	cacheWarningReported := false
	if cacheErr := memory.CacheError(); cacheErr != nil {
		reporter.LogDetailedStatus(ctx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
		cacheWarningReported = true
	}
	memory.Start()
	defer memory.CloseAndWait()

	// reading files
	discovery := services.NewFileDiscovery()

	reporter.LogDetailedStatus(ctx, "Reading source files")
	sourceDiscovery, err := discovery.DiscoverWithDiagnostics(ctx, []string{args.FindMissingSourceDir}, reporter)
	if err != nil {
		return err
	}
	reporter.LogDetailedStatus(ctx, "Reading target files")
	targetDiscovery, err := discovery.DiscoverWithDiagnostics(ctx, []string{args.FindMissingTargetDir}, reporter)
	if err != nil {
		return err
	}

	diagnostics := models.ScanDiagnostics{
		SkippedSourcePaths: append([]models.SkippedPath(nil), sourceDiscovery.SkippedPaths...),
		SkippedTargetPaths: append([]models.SkippedPath(nil), targetDiscovery.SkippedPaths...),
	}

	sourceFiles := sourceDiscovery.Files
	targetFiles := targetDiscovery.Files
	totalFiles := common.LenSyncMap(sourceFiles) + common.LenSyncMap(targetFiles)
	reporter.LogProgress(ctx, "Reading", 100)
	reporter.LogFilesCount(ctx, int64(totalFiles), int64(totalFiles))

	if err := ctx.Err(); err != nil {
		return err
	}

	// filtering read files to usefull parts for the flow
	reporter.LogProgress(ctx, "Filtering", 0)
	
	targetSizes := services.FileSizes(targetFiles)
	filteredSourceFiles := services.CopyFilesWithSizes(sourceFiles, targetSizes)
	
	sourceCandidateSizes := services.FileSizes(filteredSourceFiles)
	hashTargetMap := services.CopyFilesWithSizes(targetFiles, sourceCandidateSizes)
	candidateCount := common.LenSyncMap(filteredSourceFiles) + common.LenSyncMap(hashTargetMap)
	
	reporter.LogProgress(ctx, "Filtering", 100)
	reporter.LogFilesCount(ctx, int64(candidateCount), int64(totalFiles))

	hashErrors := make(chan error, candidateCount+1)
	sourceHashing := visuals.NewProgressTracker(ctx, reporter, "Hashing Source")
	sourceHashing.Start()
	if err := services.CreateHashes(ctx, filteredSourceFiles, args.WorkerCount, sourceHashing, memory, &hashMemory, hashErrors); err != nil {
		return err
	}
	sourceHashing.Wait()
	for _, hashErr := range drainHashErrors(hashErrors) {
		diagnostics.SkippedSourcePaths = append(diagnostics.SkippedSourcePaths, hashErr)
	}

	targetHashing := visuals.NewProgressTracker(ctx, reporter, "Hashing Target")
	targetHashing.Start()
	if err := services.CreateHashes(ctx, hashTargetMap, args.WorkerCount, targetHashing, memory, &hashMemory, hashErrors); err != nil {
		return err
	}
	targetHashing.Wait()
	for _, hashErr := range drainHashErrors(hashErrors) {
		diagnostics.SkippedTargetPaths = append(diagnostics.SkippedTargetPaths, hashErr)
	}

	memory.WaitForCache(ctx, func(completed, total int64) {
		reporter.LogProgress(ctx, "Caching", percentage(completed, total))
		reporter.LogFilesCount(ctx, completed, total)
	})
	if cacheErr := memory.CacheError(); cacheErr != nil && !cacheWarningReported {
		reporter.LogDetailedStatus(ctx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	mergeHashedInto(sourceFiles, filteredSourceFiles)
	mergeHashedInto(targetFiles, hashTargetMap)
	deleteSkippedFiles(sourceFiles, diagnostics.SkippedSourcePaths)
	deleteSkippedFiles(targetFiles, diagnostics.SkippedTargetPaths)
	publisher.PublishScanDiagnostics(diagnostics)
	reporter.LogProgress(ctx, "Comparing", 0)
	result, err := services.MatchCoverage(ctx, syncMapValues(sourceFiles), syncMapValues(targetFiles), args.ParanoidMode)
	if err != nil {
		return err
	}
	reporter.LogProgress(ctx, "Comparing", 100)
	reporter.LogFilesCount(ctx, int64(result.TotalSourceFiles), int64(result.TotalSourceFiles))
	reporter.LogProgress(ctx, "Collecting", 100)
	reporter.LogProgress(ctx, "Done", 100)
	reporter.FinishExecution(ctx)
	log.InfoWithFuncName(fmt.Sprintf("Took: %s for buffer size %d", time.Since(timer), args.BufferSizeBytes))
	publisher.PublishMissingContentResult(result)
	return nil
}

func syncMapValues(files *sync.Map) []models.FileHash {
	values := make([]models.FileHash, 0, common.LenSyncMap(files))
	files.Range(func(_, value any) bool {
		values = append(values, value.(models.FileHash))
		return true
	})
	return values
}

func mergeHashedInto(all, hashed *sync.Map) {
	hashed.Range(func(key, value any) bool {
		all.Store(key, value)
		return true
	})
}

func deleteSkippedFiles(files *sync.Map, skipped []models.SkippedPath) {
	for _, path := range skipped {
		if path.Path != "" {
			files.Delete(path.Path)
		}
	}
}

func drainHashErrors(errorsChannel <-chan error) []models.SkippedPath {
	var skipped []models.SkippedPath
	for {
		select {
		case err := <-errorsChannel:
			log.WarnWithFuncName(err.Error())
			var hashErr services.HashError
			if errors.As(err, &hashErr) {
				skipped = append(skipped, models.SkippedPath{Path: hashErr.Path, Reason: hashErr.Err.Error()})
			} else {
				skipped = append(skipped, models.SkippedPath{Reason: err.Error()})
			}
		default:
			return skipped
		}
	}
}

func percentage(current, total int64) float64 {
	if total <= 0 {
		return 100
	}
	return float64(current) / float64(total) * 100
}
