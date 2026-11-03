package services

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"path/filepath"
	"sync"
)

// FileDiscovery collects eligible files from a set of directory roots.
type FileDiscovery struct{}

type diagnosticSubmitter func(DiagnosticEvent)

// NewFileDiscovery creates the shared file-discovery component.
func NewFileDiscovery() FileDiscovery {
	return FileDiscovery{}
}

// DiscoverWithSink walks roots and submits diagnostics to the execution sink
// as soon as the walking producer encounters them.
func (d FileDiscovery) DiscoverWithSink(ctx context.Context, roots []string, reporter reporting.Reporter, sink *DiagnosticSink, handler DiagnosticHandler, scope models.DiagnosticScope) (*sync.Map, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var submitMu sync.Mutex
	var fatalMu sync.Mutex
	var fatalErr error
	submit := func(event DiagnosticEvent) {
		event.Diagnostic.Scope = scope
		submitMu.Lock()
		defer submitMu.Unlock()

		disposition := handler(event)
		sink.Submit(event, disposition)
		if disposition == ErrorFatal {
			fatalMu.Lock()
			if fatalErr == nil {
				fatalErr = event.Err
			}
			fatalMu.Unlock()
			cancel()
		}
	}

	files := &sync.Map{}
	if len(roots) == 0 {
		return files, nil
	}

	reading := visuals.NewProgressCounter(ctx, reporter, "Reading", len(roots))
	reading.Start()
	for _, root := range roots {
		go walkDirWithSink(ctx, root, files, reading, submit)
	}
	reading.Wait()

	fatalMu.Lock()
	deferredFatalErr := fatalErr
	fatalMu.Unlock()
	if deferredFatalErr != nil {
		return files, deferredFatalErr
	}
	if err := ctx.Err(); errors.Is(err, context.Canceled) {
		return files, err
	}
	return files, nil
}

func walkDirWithSink(ctx context.Context, path string, result *sync.Map, pt *visuals.ProgressCounter, submit diagnosticSubmitter) {
	defer pt.SenderFinished()

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started walking directory %s files", groupID, path))

	err := filepath.WalkDir(path, storeFilePathsWithSink(ctx, result, pt, submit))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.WarnWithFuncName(fmt.Sprintf("Group %d walking cancelled by user.", groupID))
			return
		}
		recordSkippedPath(submit, path, err)
	}
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished walking directory %s files", groupID, path))
}

func storeFilePathsWithSink(ctx context.Context, result *sync.Map, pt *visuals.ProgressCounter, submit diagnosticSubmitter) func(string, fs.DirEntry, error) error {
	return func(path string, dirEntry fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			log.WarnWithFuncName(fmt.Sprintf("Stopping file walk at %s due to context cancellation.", path))
			return ctx.Err()
		default:
		}

		if err != nil {
			recordSkippedPath(submit, path, err)
			if dirEntry != nil && dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if dirEntry == nil {
			recordSkippedPath(submit, path, fmt.Errorf("directory entry is unavailable"))
			return nil
		}

		if !dirEntry.IsDir() {
			info, err := dirEntry.Info()
			if err != nil {
				recordSkippedPath(submit, path, err)
				return nil
			}

			result.Store(path, models.FileHash{
				FileName: dirEntry.Name(),
				FilePath: path,
				FileSize: info.Size(),
				ModTime:  info.ModTime().Format(common.TimeFrmt),
			})
			pt.Channel <- 1
		}
		return nil
	}
}

func recordSkippedPath(submit diagnosticSubmitter, path string, err error) {
	if submit == nil {
		return
	}
	submit(DiagnosticEvent{
		Diagnostic: models.Diagnostic{
			Path:   path,
			Reason: err.Error(),
			Phase:  models.DiagnosticPhaseWalking,
		},
		Err:      err,
		Severity: DiagnosticSeverityWarning,
	})
}
