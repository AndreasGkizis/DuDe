package processing

import (
	"DuDe/internal/common"
	"DuDe/internal/common/fs"
	log "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	"DuDe/internal/handlers/validation"
	"DuDe/internal/reporting"

	"errors"

	"DuDe/internal/models"
	"DuDe/internal/visuals"
	"context"
	"fmt"

	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FrontendApp struct
type FrontendApp struct {
	wailsCtx context.Context
	execCtx  context.Context
	gate     *ExecutionGate
	stateMu  sync.RWMutex

	platform    string
	Args        models.ExecutionParams
	reporter    reporting.Reporter
	lastResults []models.FileHash // duplicate groups from the last completed execution
}

// NewApp creates a new App application struct
func NewApp(reporter reporting.Reporter) *FrontendApp {
	return &FrontendApp{
		reporter: reporter,
		gate:     NewExecutionGate(),
	}
}

// CancelExecution attempts to stop the currently running process.
// This function will be exposed to the Wails frontend.
func (app *FrontendApp) CancelExecution() {
	log.InfoWithFuncName("Execution cancellation requested by user.")
	app.gate.Cancel()
}

// FullReset stops any running execution, clears the cache database, and resets
// all transient application state (Args, lastResults) back to zero values.
// The Wails context, reporter, and platform are intentionally left untouched.
// A "fullReset" event is emitted so the frontend can reset its own state.
func (app *FrontendApp) FullReset() error {
	if !app.gate.BeginResetAndWait() {
		return nil
	}

	log.InfoWithFuncName("FullReset: active execution stopped before resetting state.")

	// Resolve the cache directory — mirror the resolver fallback.
	args := app.executionArgs()
	cacheDir := args.CacheDir
	if cacheDir == "" {
		cacheDir = common.GetSafeResultsDir(app.platform)
	}

	// Open the DB and truncate all cached hashes.
	// A missing or un-initialised DB is not a fatal error for a full reset.
	db, err := database.GetDatabaseConnection(cacheDir)
	if err != nil {
		log.WarnWithFuncName(fmt.Sprintf("FullReset: could not open cache DB (may not exist yet): %v", err))
	} else {
		if truncErr := database.TruncateDatabase(db); truncErr != nil {
			log.WarnWithFuncName(fmt.Sprintf("FullReset: could not truncate cache: %v", truncErr))
		}
		db.Close()
	}

	// Reset transient state only.
	app.stateMu.Lock()
	app.Args = models.ExecutionParams{}
	app.lastResults = nil
	app.stateMu.Unlock()

	app.gate.EndReset()
	runtime.EventsEmit(app.wailsCtx, "fullReset", nil)
	return nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *FrontendApp) Startup(ctx context.Context) {
	a.wailsCtx = ctx
	a.platform = runtime.Environment(a.wailsCtx).Platform
}

// CheckIfResultsExist returns true if the results JSON file is found on disk
func (a *FrontendApp) CheckIfResultsExist() bool {
	var resultsDir string
	args := a.executionArgs()

	if args.ResultsDir == "" {
		resultsDir = common.GetSafeResultsDir(a.platform)
	} else {
		resultsDir = args.ResultsDir
	}
	return ResultsFileExist(resultsDir)
}

// ShowResults opens the results file defined in the execution arguments using the default OS handler.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) ShowResults() error {
	var resultsDirectory string
	args := a.executionArgs()

	if args.ResultsDir == "" {
		resultsDirectory = common.GetSafeResultsDir(a.platform)
	} else {
		resultsDirectory = args.ResultsDir
	}

	if resultsDirectory == "" {
		a.reporter.LogDetailedStatus(a.wailsCtx, "Cannot open results: Results file path is not set.")
		return fmt.Errorf("results file path is empty")
	}

	cmd, err := common.GetOpenDirectoryFunc(resultsDirectory, a.platform)
	if err != nil {
		a.reporter.LogDetailedStatus(a.wailsCtx, fmt.Sprintf("Cannot open results: %v", err))
		runtime.EventsEmit(a.wailsCtx, "errorUpdate", err.Error())
		return fmt.Errorf("%s", err.Error())
	}

	// --- Execute Command ---
	err = cmd.Start()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute OS command to open file '%s'. Error: %v", resultsDirectory, err)
		runtime.EventsEmit(a.wailsCtx, "errorUpdate", errorMsg)
		return fmt.Errorf("failed to open file: %w", err)
	}

	return nil
}

// RevealInExplorer opens the OS file manager with the given file path highlighted/selected.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) RevealInExplorer(path string) error {
	cmd, err := common.GetOpenDirectoryFunc(common.GetFileDir(path), a.platform)
	if err != nil {
		return fmt.Errorf("unsupported platform: %s", a.platform)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to reveal file in explorer: %w", err)
	}
	return nil
}

// GetResults returns the duplicate groups found in the last completed execution.
// Each FileHash in the returned slice has DuplicatesFound populated.
// Returns nil if no execution has completed yet.
func (a *FrontendApp) GetResults() []models.FileHash {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return append([]models.FileHash(nil), a.lastResults...)
}

// SelectFolder opens a native folder selection dialog and returns the selected path.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) SelectFolder() (string, error) {
	// runtime.OpenDirectoryDialog requires the context
	windowContext := a.wailsCtx

	// Open the directory selection dialog.
	// If the user cancels, an empty string is returned, not an error.
	selectedPath, err := runtime.OpenDirectoryDialog(windowContext, runtime.OpenDialogOptions{
		Title: "Select Working Directory",
	})

	if err != nil {
		// Only returns an error if the OS failed to open the dialog
		return "", err
	}

	return selectedPath, nil
}

func (a *FrontendApp) StartExecution(args models.ExecutionParams) error {
	if a.wailsCtx == nil {
		// Safety check, though WailsInit should handle this
		return errors.New("wails application context is not initialized")
	}

	execCtx, err := a.gate.Start(a.wailsCtx)
	if err != nil {
		a.reporter.ReportError(a.wailsCtx, err.Error())
		return err
	}
	defer a.finishExecution()

	safeDir := common.GetSafeResultsDir(a.platform)

	resolver := validation.Resolver{
		V: validation.Validator{
			FS: fs.OS{},
		},
	}

	if err := resolver.ResolveAndValidateArgs(&args, safeDir); err != nil {
		executionErr := fmt.Errorf("validation failed: %w", err)
		a.reporter.ReportError(a.wailsCtx, executionErr.Error())
		return executionErr
	}

	a.stateMu.Lock()
	a.execCtx = execCtx
	a.Args = args
	a.stateMu.Unlock()

	err = runSelectedExecution(a, a.reporter)
	if errors.Is(err, context.Canceled) {
		a.reporter.LogDetailedStatus(a.wailsCtx, "Process Stopped.")
		return nil
	}
	if err != nil {
		a.reporter.ReportError(a.wailsCtx, err.Error())
	}
	return err
}

func (a *FrontendApp) finishExecution() {
	a.stateMu.Lock()
	a.execCtx = nil
	a.stateMu.Unlock()
	a.gate.Finish()
}

func (a *FrontendApp) executionArgs() models.ExecutionParams {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.Args
}

func (a *FrontendApp) executionContext() context.Context {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.execCtx
}

func (a *FrontendApp) setLastResults(results []models.FileHash) {
	a.stateMu.Lock()
	a.lastResults = results
	a.stateMu.Unlock()
}

func startExecution(app *FrontendApp, reporter reporting.Reporter) error {
	var err error
	args := app.executionArgs()
	execCtx := app.executionContext()

	log.Initialize(args.DebugMode)

	timer := time.Now()
	log.LogModelArgs(args)

	errorLogger := newExecutionErrorLogger(100)
	defer errorLogger.CloseAndWait()
	errChan := errorLogger.channel

	var senderGroups int32 = int32(len(args.Directories))

	failedCounter := 0
	mm := NewMemoryManager(&args, args.BufSize, 1)

	rt := visuals.NewProgressCounter(execCtx, app.reporter, "Reading", int(senderGroups))
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()
	cacheWarningReported := false
	if cacheErr := mm.CacheError(); cacheErr != nil {
		app.reporter.LogDetailedStatus(execCtx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
		cacheWarningReported = true
	}
	mm.Start()
	defer mm.CloseAndWait()

	var syncSourceDirFileMap sync.Map

	for _, dir := range args.Directories {
		dir := dir // capture loop variable
		go WalkDir(execCtx, dir, &syncSourceDirFileMap, rt)
	}
	rt.Wait()

	fileCount := common.LenSyncMap(&syncSourceDirFileMap)
	app.reporter.LogProgress(execCtx, "Reading", 100)
	app.reporter.LogFilesCount(execCtx, int64(fileCount), int64(fileCount))
	if fileCount == 0 {
		app.reporter.LogProgress(execCtx, "Error", 0)
		app.reporter.LogDetailedStatus(execCtx, "No files found in directory/directories! Check your paths again")
		return nil
	}

	app.reporter.LogProgress(execCtx, "Filtering", 0)
	app.reporter.LogFilesCount(execCtx, 0, int64(fileCount))
	candidateCount, skippedCount := FilterHashCandidatesBySize(&syncSourceDirFileMap)
	app.reporter.LogProgress(execCtx, "Filtering", 100)
	app.reporter.LogFilesCount(execCtx, int64(fileCount), int64(fileCount))
	log.InfoWithFuncName(fmt.Sprintf("Skipped %d files with unique sizes; %d files remain as hash candidates", skippedCount, candidateCount))
	if candidateCount == 0 {
		app.setLastResults(nil)
		app.reporter.LogDetailedStatus(execCtx, "No possible duplicates found: every file has a unique size")
		app.reporter.LogFilesCount(execCtx, int64(fileCount), int64(fileCount))
		app.reporter.LogProgress(execCtx, "Done", 100)
		app.reporter.FinishExecution(execCtx)
		return nil
	}

	pt := visuals.NewProgressTracker(execCtx, reporter, "Hashing")
	pt.Start()

	err = CreateHashes(execCtx, &syncSourceDirFileMap, args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}

	pt.Wait()
	mm.WaitForCache(execCtx, func(completed, total int64) {
		percentage := float64(completed) / float64(total) * 100
		app.reporter.LogProgress(execCtx, "Caching", percentage)
		app.reporter.LogFilesCount(execCtx, completed, total)
	})
	if cacheErr := mm.CacheError(); cacheErr != nil && !cacheWarningReported {
		app.reporter.LogDetailedStatus(execCtx, fmt.Sprintf("Cache unavailable; continuing without cache: %v", cacheErr))
	}

	findTracker := visuals.NewProgressTracker(execCtx, reporter, "Finding")
	findTracker.Start()

	FindDuplicatesInMap(execCtx, &syncSourceDirFileMap, findTracker)

	findTracker.Wait()

	length := common.LenSyncMap(&syncSourceDirFileMap)

	log.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 && args.ParanoidMode {
		compareTracker := visuals.NewProgressTracker(execCtx, reporter, "Comparing")
		compareTracker.Start()

		compareErr := EnsureDuplicates(execCtx, &syncSourceDirFileMap, compareTracker, args.CPUs)
		compareTracker.Wait()
		if compareErr != nil {
			app.setLastResults(nil)
			return fmt.Errorf("verify duplicates: %w", compareErr)
		}
	}

	// Collect verified duplicate groups and cache them for GetResults()
	groupsToCollect := common.LenSyncMap(&syncSourceDirFileMap)
	app.reporter.LogProgress(execCtx, "Collecting", 0)
	app.reporter.LogFilesCount(execCtx, 0, int64(groupsToCollect))
	var groups []models.FileHash
	collectedGroups := 0
	syncSourceDirFileMap.Range(func(_, v any) bool {
		if fh, ok := v.(models.FileHash); ok && len(fh.DuplicatesFound) > 0 {
			groups = append(groups, fh)
		}
		collectedGroups++
		return true
	})
	app.setLastResults(groups)
	app.reporter.LogProgress(execCtx, "Collecting", 100)
	app.reporter.LogFilesCount(execCtx, int64(collectedGroups), int64(groupsToCollect))

	length = common.LenSyncMap(&syncSourceDirFileMap)
	if length != 0 {
		timer1 := time.Now()

		err = SaveResultsAsCSV(&syncSourceDirFileMap, args.ResultsDir)
		if err != nil {
			log.FatalWithFuncName(fmt.Sprintf("Error saving result: %v", err))
			return err
		}

		log.InfoWithFuncName(fmt.Sprintf("Took: %s to look through bytes", time.Since(timer1)))
	} else {
		log.InfoWithFuncName("No duplicates were found")
	}

	log.InfoWithFuncName(fmt.Sprintf("Took: %s for buffer size %d", time.Since(timer), args.BufSize))
	log.InfoWithFuncName(fmt.Sprintf("Failed %d times to send to memoryChan", failedCounter))
	app.reporter.LogFilesCount(app.wailsCtx, int64(fileCount), int64(fileCount))
	app.reporter.LogProgress(app.wailsCtx, "Done", 100)
	app.reporter.FinishExecution(app.wailsCtx)

	return nil
}

type executionErrorLogger struct {
	channel chan error
	wg      sync.WaitGroup
	once    sync.Once
}

func newExecutionErrorLogger(bufferSize int) *executionErrorLogger {
	logger := &executionErrorLogger{
		channel: make(chan error, bufferSize),
	}
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
	logger.once.Do(func() {
		close(logger.channel)
	})
	logger.wg.Wait()
}
