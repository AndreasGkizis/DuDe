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
func (flow FindMissingContentFlow) Run(ctx context.Context, args models.ExecutionParams, reporter reporting.Reporter, publisher ResultPublisher, options RunOptions) error {
	//start-up
	log.Initialize(args.DebugMode)
	timer := time.Now()
	log.LogModelArgs(args)
	if options.HandleDiagnostic == nil {
		return errors.New("diagnostic handler is required")
	}
	sink := services.NewDiagnosticSink()
	defer sink.Close()

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
	sourceFiles, err := discovery.DiscoverWithSink(ctx, []string{args.FindMissingSourceDir}, reporter, sink, options.HandleDiagnostic, models.DiagnosticScopeSource)
	if err != nil {
		sink.Close()
		return err
	}
	reporter.LogDetailedStatus(ctx, "Reading target files")
	targetFiles, err := discovery.DiscoverWithSink(ctx, []string{args.FindMissingTargetDir}, reporter, sink, options.HandleDiagnostic, models.DiagnosticScopeTarget)
	if err != nil {
		sink.Close()
		return err
	}

	diagnostics := models.NewScanDiagnostics(models.DiagnosticStatusCompleted)
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

	sourceHashing := visuals.NewProgressTracker(ctx, reporter, "Hashing Source")
	sourceHashing.Start()
	if err := services.CreateHashes(ctx, filteredSourceFiles, args.WorkerCount, sourceHashing, memory, &hashMemory, sink, options.HandleDiagnostic, models.DiagnosticScopeSource); err != nil {
		sink.Close()
		return err
	}
	sourceHashing.Wait()

	targetHashing := visuals.NewProgressTracker(ctx, reporter, "Hashing Target")
	targetHashing.Start()
	if err := services.CreateHashes(ctx, hashTargetMap, args.WorkerCount, targetHashing, memory, &hashMemory, sink, options.HandleDiagnostic, models.DiagnosticScopeTarget); err != nil {
		sink.Close()
		return err
	}
	targetHashing.Wait()

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
	sink.Close()
	diagnostics = completedDiagnosticsFromSnapshot(sink.Snapshot())
	deleteSkippedFiles(sourceFiles, diagnostics.Diagnostics, models.DiagnosticScopeSource)
	deleteSkippedFiles(targetFiles, diagnostics.Diagnostics, models.DiagnosticScopeTarget)
	publisher.PublishScanDiagnostics(diagnostics)
	reporter.LogProgress(ctx, "Comparing", 0)
	result, err := services.MatchCoverage(ctx, syncMapValues(sourceFiles), syncMapValues(targetFiles), args.ParanoidMode)
	if err != nil {
		return err
	}
	reporter.LogProgress(ctx, "Comparing", 100)
	reporter.LogFilesCount(ctx, int64(result.TotalSourceFiles), int64(result.TotalSourceFiles))
	reporter.LogProgress(ctx, "Collecting", 100)
	reporter.LogProgress(ctx, "Reporting", 0)
	reportPath, err := services.SaveCoverageReport(result, diagnostics, args.ResultsDir)
	if err != nil {
		return err
	}
	result.ReportPath = reportPath
	result.Diagnostics = diagnostics
	reporter.LogProgress(ctx, "Reporting", 100)
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

func deleteSkippedFiles(files *sync.Map, diagnostics []models.Diagnostic, scope models.DiagnosticScope) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Scope == scope && diagnostic.Path != "" {
			files.Delete(diagnostic.Path)
		}
	}
}

func drainHashErrors(errorsChannel <-chan error, scope models.DiagnosticScope) []models.Diagnostic {
	var skipped []models.Diagnostic
	for {
		select {
		case err := <-errorsChannel:
			log.WarnWithFuncName(err.Error())
			var hashErr services.HashError
			if errors.As(err, &hashErr) {
				skipped = append(skipped, models.Diagnostic{Path: hashErr.Path, Reason: hashErr.Err.Error(), Phase: models.DiagnosticPhaseHashing, Scope: scope})
			} else {
				skipped = append(skipped, models.Diagnostic{Reason: err.Error(), Phase: models.DiagnosticPhaseHashing, Scope: scope})
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
