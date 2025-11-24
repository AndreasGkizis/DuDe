package processing

import (
	common "DuDe-wails/internal/common"
	logger "DuDe-wails/internal/common/logger"
	db "DuDe-wails/internal/db"
	models "DuDe-wails/internal/models"
	reporting "DuDe-wails/internal/reporting"
	visuals "DuDe-wails/internal/visuals"

	// visuals "DuDe-wails/internal/visuals"
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

// Greet returns a greeting for the given name
func (a *FrontendApp) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
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
	a.Args = args
	return startExecution(args, a)
}

func startExecution(args models.ExecutionParams, reporter reporting.Reporter) error {
	log := logger.Logger
	timer := time.Now()

	Args := args

	// visuals.Intro()

	db, err := db.NewDatabase(Args.CacheDir)
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
	if Args.DualFolderModeEnabled {
		senderGroups = 2
	} else {
		senderGroups = 1
	}

	failedCounter := 0
	mm := NewMemoryManager(db, Args.BufSize, 1)
	mm.Start()
	rt := visuals.NewProgressCounter("Reading", int(senderGroups), reporter)
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	var syncSourceDirFileMap sync.Map

	go WalkDir(Args.SourceDir, &syncSourceDirFileMap, rt)

	if Args.DualFolderModeEnabled {
		go WalkDir(Args.TargetDir, &syncSourceDirFileMap, rt)
	}
	rt.Wait()

	// len := common.LenSyncMap(&syncSourceDirFileMap)
	// // if len == 0 {
	// // 	visuals.EmptyDir("asd")
	// // }

	pt := visuals.NewProgressTracker("Hashing", reporter)
	pt.Start(50)

	err = CreateHashes(&syncSourceDirFileMap, Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		return err
	}

	pt.Wait()
	mm.Wait()
	close(errChan)

	findTracker := visuals.NewProgressTracker("Finding", reporter)
	findTracker.Start(50)

	FindDuplicatesInMap(&syncSourceDirFileMap, findTracker)

	findTracker.Wait()
	length := common.LenSyncMap(&syncSourceDirFileMap)

	logger.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 {
		timer1 := time.Now()

		if Args.ParanoidMode {
			compareTracker := visuals.NewProgressTracker("Comparing", reporter)
			compareTracker.Start(50)

			EnsureDuplicates(&syncSourceDirFileMap, compareTracker, Args.CPUs)

			compareTracker.Wait()
		}

		flattenedDuplicates := GetFlattened(&syncSourceDirFileMap)
		err = SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)
		if err != nil {
			log.Fatalf("Error saving result: %v", err)
			return err

		}

		log.Infof("Took: %s to look through bytes", time.Since(timer1))
	} else {
		log.Info("No duplicates were found")
	}

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memoryChan", failedCounter)
	reporter.LogProgress("Done", 100)

	// visuals.Outro()
	return nil
}
