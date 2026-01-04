package processing

import (
	"DuDe/internal/common"
	"DuDe/internal/common/fs"
	"DuDe/internal/common/logger"
	"DuDe/internal/db"
	"DuDe/internal/handlers/validation"
	"DuDe/internal/reporting"
	"errors"

	"DuDe/internal/models"
	"DuDe/internal/visuals"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FrontendApp struct
type FrontendApp struct {
	// PERMANENT: Wails Context (Set once in WailsInit)
	wailsCtx context.Context
	// TEMPORARY: Execution Context (Set in StartExecution, Cleared in defer)
	cancelFunc context.CancelFunc
	execCtx    context.Context
	Args       models.ExecutionParams
	reporter   reporting.Reporter
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
		logger.Logger.Info("Execution cancellation requested by user.")
		app.cancelFunc()
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *FrontendApp) Startup(ctx context.Context) {
	a.wailsCtx = ctx
}

// ShowResults opens the results file defined in the execution arguments using the default OS handler.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) ShowResults() error {
	resultsFilePath := filepath.Join(a.Args.ResultsDir, common.ResFilename)

	if resultsFilePath == "" {
		a.reporter.LogDetailedStatus(a.wailsCtx, "Cannot open results: Results file path is not set.")
		return fmt.Errorf("results file path is empty")
	}

	var cmd *exec.Cmd

	// --- Determine OS-Specific Command ---
	switch runtime.Environment(a.wailsCtx).Platform {
	case "windows":
		// Windows: uses 'start' command, which must be run via cmd.exe /C
		cmd = exec.Command("cmd", "/C", "start", "", resultsFilePath)
	case "darwin":
		// macOS: uses 'open' command
		cmd = exec.Command("open", resultsFilePath)
	case "linux":
		// Linux: uses 'xdg-open'
		cmd = exec.Command("xdg-open", resultsFilePath)
	default:
		errorMsg := fmt.Sprintf("Unsupported operating system: %s", runtime.Environment(a.wailsCtx).Platform)
		runtime.EventsEmit(a.wailsCtx, "errorUpdate", errorMsg)
		return fmt.Errorf("%s", errorMsg)
	}

	// --- Execute Command ---
	err := cmd.Start()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute OS command to open file '%s'. Error: %v", resultsFilePath, err)
		runtime.EventsEmit(a.wailsCtx, "errorUpdate", errorMsg)
		return fmt.Errorf("failed to open file: %w", err)
	}

	return nil
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
	executableDir := common.GetExecutableDir()

	resolver := validation.Resolver{
		V: validation.Validator{
			FS: fs.OS{},
		},
	}

	if err := resolver.ResolveAndValidateArgs(&args, executableDir); err != nil {
		// Log the failure to the frontend
		a.reporter.LogDetailedStatus(a.wailsCtx, fmt.Sprintf("Argument Validation Failed: %v", err))
		// Throw an error back to the frontend to stop execution
		return fmt.Errorf("validation failed: %w", err)
	}

	a.Args = args
	return startExecution(a, a.reporter)
}

func startExecution(app *FrontendApp, reporter reporting.Reporter) error {
	// Ensure cleanup of stored context when execution finishes normally
	defer func() {
		if app.cancelFunc != nil {
			app.cancelFunc()
			app.cancelFunc = nil
		}
	}()

	log := logger.Logger
	timer := time.Now()
	logger.LogModelArgs(app.Args)

	db, err := db.NewDatabase(app.Args.CacheDir)
	if err != nil {
		logger.ErrorWithFuncName(err.Error())
	}

	errChan := make(chan error, 100)
	go func() {
		for err := range errChan {
			logger.WarnWithFuncName(err.Error())
		}
	}()

	var senderGroups int32
	if app.Args.DualFolderModeEnabled {
		senderGroups = 2
	} else {
		senderGroups = 1
	}

	failedCounter := 0
	mm := NewMemoryManager(db, app.Args.BufSize, 1)
	mm.Start()
	rt := visuals.NewProgressCounter(app.execCtx, app.reporter, "Reading", int(senderGroups))
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	var syncSourceDirFileMap sync.Map

	go WalkDir(app.execCtx, app.Args.SourceDir, &syncSourceDirFileMap, rt)

	if app.Args.DualFolderModeEnabled {
		go WalkDir(app.execCtx, app.Args.TargetDir, &syncSourceDirFileMap, rt)
	}
	rt.WaitForSenders()

	len := common.LenSyncMap(&syncSourceDirFileMap)
	if len == 0 {

		app.reporter.LogProgress(app.execCtx, "Error", 0)

		return ErrNoFilesFound
	}

	pt := visuals.NewProgressTracker(app.execCtx, reporter, "Hashing")
	pt.Start()

	err = CreateHashes(app.execCtx, &syncSourceDirFileMap, app.Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}

	pt.Wait()
	mm.Wait()
	close(errChan)

	findTracker := visuals.NewProgressTracker(app.execCtx, reporter, "Finding")
	findTracker.Start()

	FindDuplicatesInMap(app.execCtx, &syncSourceDirFileMap, findTracker)

	findTracker.Wait()
	length := common.LenSyncMap(&syncSourceDirFileMap)

	logger.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 {
		timer1 := time.Now()

		if app.Args.ParanoidMode {
			compareTracker := visuals.NewProgressTracker(app.execCtx, reporter, "Comparing")
			compareTracker.Start()

			EnsureDuplicates(app.execCtx, &syncSourceDirFileMap, compareTracker, app.Args.CPUs)

			compareTracker.Wait()
		}

		flattenedDuplicates := GetFlattened(&syncSourceDirFileMap)
		err = SaveResultsAsCSV(flattenedDuplicates, app.Args.ResultsDir)
		if err != nil {
			log.Fatalf("Error saving result: %v", err)
			return err
		}

		log.Infof("Took: %s to look through bytes", time.Since(timer1))
	} else {
		log.Info("No duplicates were found")
	}

	log.Infof("Took: %s for buffer size %d", time.Since(timer), app.Args.BufSize)
	log.Infof("Failed %d times to send to memoryChan", failedCounter)
	app.reporter.LogProgress(app.wailsCtx, "Done", 100)

	return nil
}
