package processing

import (
	"DuDe/internal/common"
	"DuDe/internal/common/fs"
	log "DuDe/internal/common/logger"
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
	wailsCtx   context.Context    // PERMANENT: Wails Context (Set once in WailsInit)
	cancelFunc context.CancelFunc // TEMPORARY: Execution Context (Set in StartExecution, Cleared in defer)

	platform    string
	execCtx     context.Context
	Args        models.ExecutionParams
	reporter    reporting.Reporter
	lastResults []models.FileHash // duplicate groups from the last completed execution
}

// NewApp creates a new App application struct
func NewApp(reporter reporting.Reporter) *FrontendApp {
	return &FrontendApp{
		reporter: reporter,
	}
}

// CancelExecution attempts to stop the currently running process.
// This function will be exposed to the Wails frontend.
func (app *FrontendApp) CancelExecution() {
	if app.cancelFunc != nil {
		log.InfoWithFuncName("Execution cancellation requested by user.")
		app.cancelFunc()
	}
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

	if a.Args.ResultsDir == "" {
		resultsDir = common.GetSafeResultsDir(a.platform)
	} else {
		resultsDir = a.Args.ResultsDir
	}
	return ResultsFileExist(resultsDir)
}

// ShowResults opens the results file defined in the execution arguments using the default OS handler.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) ShowResults() error {
	var resultsDirectory string

	if a.Args.ResultsDir == "" {
		resultsDirectory = common.GetSafeResultsDir(a.platform)
	} else {
		resultsDirectory = a.Args.ResultsDir
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
	return a.lastResults
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
	a.execCtx, a.cancelFunc = context.WithCancel(a.wailsCtx)
	safeDir := common.GetSafeResultsDir(a.platform)

	resolver := validation.Resolver{
		V: validation.Validator{
			FS: fs.OS{},
		},
	}

	if err := resolver.ResolveAndValidateArgs(&args, safeDir); err != nil {
		// Log the failure to the frontend
		a.reporter.LogDetailedStatus(a.wailsCtx, fmt.Sprintf("Argument Validation Failed: %v", err))
		// Throw an error back to the frontend to stop execution
		return fmt.Errorf("validation failed: %w", err)
	}

	a.Args = args

	// DEV: simulate execution - remove before shipping
	// go simulateExecution(a.execCtx, a.reporter)
	// return nil

	return startExecution(a, a.reporter)
}

func startExecution(app *FrontendApp, reporter reporting.Reporter) error {
	var err error

	// Ensure cleanup of stored context when execution finishes normally
	defer func() {
		if app.cancelFunc != nil {
			app.cancelFunc()
			app.cancelFunc = nil
		}
	}()
	log.Initialize(app.Args.DebugMode)

	timer := time.Now()
	log.LogModelArgs(app.Args)

	errChan := make(chan error, 100)
	go func() {
		for err := range errChan {
			log.WarnWithFuncName(err.Error())
		}
	}()

	var senderGroups int32 = int32(len(app.Args.Directories))

	failedCounter := 0
	mm := NewMemoryManager(&app.Args, app.Args.BufSize, 1)
	mm.Start()

	rt := visuals.NewProgressCounter(app.execCtx, app.reporter, "Reading", int(senderGroups))
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	var syncSourceDirFileMap sync.Map

	for _, dir := range app.Args.Directories {
		dir := dir // capture loop variable
		go WalkDir(app.execCtx, dir, &syncSourceDirFileMap, rt)
	}
	rt.WaitForSenders()

	fileCount := common.LenSyncMap(&syncSourceDirFileMap)
	if fileCount == 0 {
		app.reporter.LogProgress(app.execCtx, "Error", 0)
		app.reporter.LogDetailedStatus(app.execCtx, "No files found in directory/directories! Check your paths again")
		return nil
	}

	pt := visuals.NewProgressTracker(app.execCtx, reporter, "Hashing")
	pt.Start()

	err = CreateHashes(app.execCtx, &syncSourceDirFileMap, app.Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}

	pt.Wait()
	mm.Wait()

	close(errChan)

	findTracker := visuals.NewProgressTracker(app.execCtx, reporter, "Finding")
	findTracker.Start()

	FindDuplicatesInMap(app.execCtx, &syncSourceDirFileMap, findTracker)

	findTracker.Wait()

	// Collect duplicate groups and cache them for GetResults()
	var groups []models.FileHash
	syncSourceDirFileMap.Range(func(_, v any) bool {
		if fh, ok := v.(models.FileHash); ok && len(fh.DuplicatesFound) > 0 {
			groups = append(groups, fh)
		}
		return true
	})
	app.lastResults = groups

	length := common.LenSyncMap(&syncSourceDirFileMap)

	log.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 {
		timer1 := time.Now()

		if app.Args.ParanoidMode {
			compareTracker := visuals.NewProgressTracker(app.execCtx, reporter, "Comparing")
			compareTracker.Start()

			EnsureDuplicates(app.execCtx, &syncSourceDirFileMap, compareTracker, app.Args.CPUs)

			compareTracker.Wait()
		}

		err = SaveResultsAsCSV(&syncSourceDirFileMap, app.Args.ResultsDir)
		if err != nil {
			log.FatalWithFuncName(fmt.Sprintf("Error saving result: %v", err))
			return err
		}

		log.InfoWithFuncName(fmt.Sprintf("Took: %s to look through bytes", time.Since(timer1)))
	} else {
		log.InfoWithFuncName("No duplicates were found")
	}

	log.InfoWithFuncName(fmt.Sprintf("Took: %s for buffer size %d", time.Since(timer), app.Args.BufSize))
	log.InfoWithFuncName(fmt.Sprintf("Failed %d times to send to memoryChan", failedCounter))
	app.reporter.LogProgress(app.wailsCtx, "Done", 100)
	app.reporter.FinishExecution(app.wailsCtx)

	return nil
}
