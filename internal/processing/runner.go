package processing

import (
	"DuDe/internal/common"
	"DuDe/internal/common/fs"
	log "DuDe/internal/common/logger"
	database "DuDe/internal/db"
	"DuDe/internal/handlers/validation"
	"DuDe/internal/processing/flows"
	"DuDe/internal/reporting"

	"errors"

	"DuDe/internal/models"
	"context"
	"fmt"
	"os"

	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FrontendApp struct
type executionState struct {
	DuplicateResult      *models.DuplicateResult
	MissingContentResult *models.MissingContentResult
	ScanDiagnostics      models.ScanDiagnostics
}

type FrontendApp struct {
	wailsCtx context.Context
	execCtx  context.Context
	gate     *ExecutionGate
	stateMu  sync.RWMutex

	platform       string
	Args           models.ExecutionParams
	reporter       reporting.Reporter
	pathValidator  validation.PathValidator
	flowRegistry   *flows.Registry
	executionState executionState
}

// NewApp creates a new App application struct
func NewApp(reporter reporting.Reporter) *FrontendApp {
	pathValidator := validation.FilesystemValidator{
		FS: fs.OS{},
	}

	app := &FrontendApp{
		reporter:       reporter,
		pathValidator:  pathValidator,
		gate:           NewExecutionGate(),
		executionState: executionState{ScanDiagnostics: models.NewScanDiagnostics(models.DiagnosticStatusNotStarted)},
	}

	duplicateFlow := flows.NewDuplicateFlow(pathValidator)
	missingContentFlow := flows.NewFindMissingContentFlow(pathValidator)
	flowRegistry, err := flows.NewRegistry(duplicateFlow, missingContentFlow)
	if err != nil {
		panic(fmt.Sprintf("create execution flow registry: %v", err))
	}

	app.flowRegistry = flowRegistry
	return app
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
	app.executionState = executionState{ScanDiagnostics: models.NewScanDiagnostics(models.DiagnosticStatusNotStarted)}
	app.stateMu.Unlock()

	app.gate.EndReset()
	// Tests and headless callers may provide a plain context rather than the
	// Wails lifecycle context. There is no frontend to notify in that case.
	if app.wailsCtx != nil && app.wailsCtx.Value("frontend") != nil {
		runtime.EventsEmit(app.wailsCtx, "fullReset", nil)
	}
	return nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *FrontendApp) Startup(ctx context.Context) {
	a.wailsCtx = ctx
	a.platform = runtime.Environment(a.wailsCtx).Platform
}

// CheckIfResultsExist returns true when the current report exists on disk.
func (a *FrontendApp) CheckIfResultsExist() bool {
	a.stateMu.RLock()
	var reportPath string
	if a.executionState.DuplicateResult != nil {
		reportPath = a.executionState.DuplicateResult.ReportPath
	} else if a.executionState.MissingContentResult != nil {
		reportPath = a.executionState.MissingContentResult.ReportPath
	}
	a.stateMu.RUnlock()

	if reportPath == "" {
		return false
	}
	info, err := os.Stat(reportPath)
	return err == nil && info.Mode().IsRegular()
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

// GetResults returns the last completed duplicate result, or nil when no
// completed duplicate execution exists.
func (a *FrontendApp) GetResults() *models.DuplicateResult {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return copyDuplicateResult(a.executionState.DuplicateResult)
}

// GetMissingContentResult returns the last completed coverage result.
func (a *FrontendApp) GetMissingContentResult() *models.MissingContentResult {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return copyMissingContentResult(a.executionState.MissingContentResult)
}

// GetScanDiagnostics returns the current execution diagnostics state.
func (a *FrontendApp) GetScanDiagnostics() *models.ScanDiagnostics {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	result := copyScanDiagnostics(a.executionState.ScanDiagnostics)
	return &result
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

	// Resolve and validate before acquiring the execution gate. Validation is
	// mode-specific, and a rejected request must not briefly occupy the active
	// execution slot or interfere with FullReset.
	safeDir := common.GetSafeResultsDir(a.platform)
	flow, err := a.flowRegistry.Resolve(args.Mode)
	if err != nil {
		a.reporter.ReportError(a.wailsCtx, err.Error())
		return err
	}

	validation.ResolveExecutionArgs(&args, safeDir)
	if err := validation.ValidateSharedExecutionArgs(&args, a.pathValidator); err != nil {
		executionErr := fmt.Errorf("validation failed: %w", err)
		a.reporter.ReportError(a.wailsCtx, executionErr.Error())
		return executionErr
	}
	if err := flow.Validate(&args); err != nil {
		executionErr := fmt.Errorf("validation failed: %w", err)
		a.reporter.ReportError(a.wailsCtx, executionErr.Error())
		return executionErr
	}

	execCtx, err := a.gate.Start(a.wailsCtx)
	if err != nil {
		a.reporter.ReportError(a.wailsCtx, err.Error())
		return err
	}
	defer a.finishExecution()

	a.stateMu.Lock()
	a.execCtx = execCtx
	a.Args = args
	a.executionState = executionState{ScanDiagnostics: models.NewScanDiagnostics(models.DiagnosticStatusRunning)}
	a.stateMu.Unlock()

	// Flows report their concrete result to this collector. FrontendApp only
	// publishes it to synchronized application state after Run returns
	// successfully, so failed or cancelled executions cannot publish results.
	result := &executionResultPublisher{}
	err = flow.Run(execCtx, args, a.reporter, result, flows.RunOptions{HandleDiagnostic: flows.DefaultDiagnosticHandler})
	if errors.Is(err, context.Canceled) {
		a.publishExecutionState(executionState{ScanDiagnostics: models.NewScanDiagnostics(models.DiagnosticStatusCancelled)})
		a.reporter.LogDetailedStatus(a.wailsCtx, "Process Stopped.")
		return nil
	}
	if err != nil {
		a.publishExecutionState(executionState{ScanDiagnostics: models.NewScanDiagnostics(models.DiagnosticStatusInconclusive)})
		a.reporter.ReportError(a.wailsCtx, err.Error())
		return err
	}

	result.PublishTo(a)
	a.reporter.FinishExecution(a.wailsCtx)
	return nil
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

// executionResultPublisher collects concrete flow results until the flow has
// completed successfully. It deliberately does not expose a shared result
// envelope.
type executionResultPublisher struct {
	duplicate      *models.DuplicateResult
	missingContent *models.MissingContentResult
}

func (p *executionResultPublisher) PublishDuplicateResult(result models.DuplicateResult) {
	copyResult := result
	copyResult.Groups = append([]models.FileHash(nil), result.Groups...)
	copyResult.Diagnostics = copyScanDiagnostics(result.Diagnostics)
	p.duplicate = &copyResult
}

func (p *executionResultPublisher) PublishMissingContentResult(result models.MissingContentResult) {
	copyResult := result
	copyResult.MissingFiles = append([]models.MissingSourceFile(nil), result.MissingFiles...)
	copyResult.Diagnostics = copyScanDiagnostics(result.Diagnostics)
	p.missingContent = &copyResult
}

func (p *executionResultPublisher) PublishTo(app *FrontendApp) {
	if p.duplicate != nil {
		result := copyDuplicateResult(p.duplicate)
		app.publishExecutionState(executionState{DuplicateResult: result, ScanDiagnostics: result.Diagnostics})
		return
	}
	if p.missingContent != nil {
		result := copyMissingContentResult(p.missingContent)
		app.publishExecutionState(executionState{MissingContentResult: result, ScanDiagnostics: result.Diagnostics})
	}
}

func (a *FrontendApp) publishExecutionState(state executionState) {
	state.ScanDiagnostics.Diagnostics = append([]models.Diagnostic{}, state.ScanDiagnostics.Diagnostics...)
	a.stateMu.Lock()
	a.executionState = state
	a.stateMu.Unlock()
}

func copyScanDiagnostics(source models.ScanDiagnostics) models.ScanDiagnostics {
	result := source
	result.Diagnostics = append([]models.Diagnostic{}, source.Diagnostics...)
	if result.Diagnostics == nil {
		result.Diagnostics = []models.Diagnostic{}
	}
	return result
}

func copyDuplicateResult(source *models.DuplicateResult) *models.DuplicateResult {
	if source == nil {
		return nil
	}
	result := *source
	result.Groups = append([]models.FileHash(nil), source.Groups...)
	result.Diagnostics = copyScanDiagnostics(source.Diagnostics)
	return &result
}

func copyMissingContentResult(source *models.MissingContentResult) *models.MissingContentResult {
	if source == nil {
		return nil
	}
	result := *source
	result.MissingFiles = append([]models.MissingSourceFile(nil), source.MissingFiles...)
	result.Diagnostics = copyScanDiagnostics(source.Diagnostics)
	return &result
}
