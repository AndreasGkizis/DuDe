package processing

import (
	common "DuDe/internal/common"
	logger "DuDe/internal/common/logger"
	db "DuDe/internal/db"
	"DuDe/internal/handlers"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
	"os/exec"
	"path/filepath"

	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FrontendApp struct
type FrontendApp struct {
	ctx  context.Context
	Args models.ExecutionParams
}

// NewApp creates a new App application struct
func NewApp() *FrontendApp {
	return &FrontendApp{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *FrontendApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

// ShowResults opens the results file defined in the execution arguments using the default OS handler.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) ShowResults() error {
	resultsFilePath := filepath.Join(a.Args.ResultsDir, common.ResFilename)

	if resultsFilePath == "" {
		a.LogDetailedStatus("Cannot open results: Results file path is not set.")
		return fmt.Errorf("results file path is empty")
	}

	var cmd *exec.Cmd

	// --- Determine OS-Specific Command ---
	switch runtime.Environment(a.ctx).Platform {
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
		errorMsg := fmt.Sprintf("Unsupported operating system: %s", runtime.Environment(a.ctx).Platform)
		runtime.EventsEmit(a.ctx, "errorUpdate", errorMsg)
		return fmt.Errorf(errorMsg)
	}

	// --- Execute Command ---
	err := cmd.Start()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute OS command to open file '%s'. Error: %v", resultsFilePath, err)
		runtime.EventsEmit(a.ctx, "errorUpdate", errorMsg)
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Optional: Log success
	a.LogDetailedStatus(fmt.Sprintf("Successfully requested opening of file: %s", resultsFilePath))
	return nil
}

// SelectFolder opens a native folder selection dialog and returns the selected path.
// It is directly exposed to the JavaScript frontend.
func (a *FrontendApp) SelectFolder() (string, error) {
	// runtime.OpenDirectoryDialog requires the context
	windowContext := a.ctx

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

type ProgressUpdate struct {
	Title   string `json:"title"`
	Percent int    `json:"percent"`
}

// LogDetailedStatus sends continuous log messages to the detailed status box.
func (a *FrontendApp) LogDetailedStatus(message string) {
	// Use a new event name specifically for detailed logging
	runtime.EventsEmit(a.ctx, "detailedLog", message)
}

// LogProgress sends progress percentage and title to the frontend.
func (a *FrontendApp) LogProgress(title string, percent int) {
	update := ProgressUpdate{
		Title:   title,
		Percent: percent,
	}
	// Wails automatically marshals the struct to JSON
	runtime.EventsEmit(a.ctx, "progressUpdate", update)
}

func (a *FrontendApp) StartExecution(args models.ExecutionParams) error {
	if err := handlers.ResolveAndValidateArgs(&args); err != nil {
		// Log the failure to the frontend
		a.LogDetailedStatus(fmt.Sprintf("Argument Validation Failed: %v", err))
		// Throw an error back to the frontend to stop execution
		return fmt.Errorf("validation failed: %w", err)
	}

	a.Args = args
	return startExecution(a)
}

func startExecution(app *FrontendApp) error {
	log := logger.Logger
	timer := time.Now()

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
	rt := visuals.NewProgressCounter("Reading", int(senderGroups), app)
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	var syncSourceDirFileMap sync.Map

	go WalkDir(app.Args.SourceDir, &syncSourceDirFileMap, rt)

	if app.Args.DualFolderModeEnabled {
		go WalkDir(app.Args.TargetDir, &syncSourceDirFileMap, rt)
	}
	rt.Wait()

	pt := visuals.NewProgressTracker("Hashing", app)
	pt.Start(50)

	err = CreateHashes(&syncSourceDirFileMap, app.Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}

	pt.Wait()
	mm.Wait()
	close(errChan)

	findTracker := visuals.NewProgressTracker("Finding", app)
	findTracker.Start(50)

	FindDuplicatesInMap(&syncSourceDirFileMap, findTracker)

	findTracker.Wait()
	length := common.LenSyncMap(&syncSourceDirFileMap)

	logger.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 {
		timer1 := time.Now()

		if app.Args.ParanoidMode {
			compareTracker := visuals.NewProgressTracker("Comparing", app)
			compareTracker.Start(50)

			EnsureDuplicates(&syncSourceDirFileMap, compareTracker, app.Args.CPUs)

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
	app.LogProgress("Done", 100)

	// visuals.Outro()
	return nil
}
