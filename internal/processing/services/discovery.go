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
	"os"
	"path/filepath"
	"sync"
)

// FileDiscovery collects eligible files from a set of directory roots. The
// underlying walk remains the same as the duplicate flow's existing walk so
// this extraction does not change filesystem eligibility or error handling.
type FileDiscovery struct{}

// DiscoveryResult contains discovered files and non-fatal entries skipped by
// the walker.
type DiscoveryResult struct {
	Files        *sync.Map
	SkippedPaths []models.SkippedPath
}

// NewFileDiscovery creates the shared file-discovery component.
func NewFileDiscovery() FileDiscovery {
	return FileDiscovery{}
}

// Discover walks each root concurrently and returns the collected file
// metadata. Directory-walk errors are reported by WalkDir as before; only
// cancellation is returned to the caller.
func (d FileDiscovery) Discover(ctx context.Context, roots []string, reporter reporting.Reporter) (*sync.Map, error) {
	result, err := d.DiscoverWithDiagnostics(ctx, roots, reporter)
	return result.Files, err
}

// DiscoverWithDiagnostics walks roots using the duplicate-mode discovery rules
// and retains non-fatal skipped-entry diagnostics for callers that need them.
func (FileDiscovery) DiscoverWithDiagnostics(ctx context.Context, roots []string, reporter reporting.Reporter) (DiscoveryResult, error) {
	result := DiscoveryResult{Files: &sync.Map{}}
	if len(roots) == 0 {
		return result, nil
	}

	reading := visuals.NewProgressCounter(ctx, reporter, "Reading", len(roots))
	reading.Start()
	var skippedMu sync.Mutex
	for _, root := range roots {
		go walkDirWithDiagnostics(ctx, root, result.Files, reading, &result.SkippedPaths, &skippedMu)
	}
	reading.Wait()

	if err := ctx.Err(); errors.Is(err, context.Canceled) {
		return result, err
	}
	return result, nil
}

func WalkDir(ctx context.Context, path string, result *sync.Map, pt *visuals.ProgressCounter) {
	walkDirWithDiagnostics(ctx, path, result, pt, nil, nil)
}

func walkDirWithDiagnostics(ctx context.Context, path string, result *sync.Map, pt *visuals.ProgressCounter, skipped *[]models.SkippedPath, skippedMu *sync.Mutex) {
	defer func() {
		pt.SenderFinished()
	}()

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started walking directory %s files", groupID, path))

	err := filepath.WalkDir(path, storeFilePathsWithDiagnostics(ctx, result, pt, skipped, skippedMu))
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.WarnWithFuncName(fmt.Sprintf("Group %d walking cancelled by user.", groupID))
			return
		}
		log.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished walking directory %s files", groupID, path))
}

func storeFilePaths(ctx context.Context, result *sync.Map, pt *visuals.ProgressCounter) func(string, fs.DirEntry, error) error {
	return storeFilePathsWithDiagnostics(ctx, result, pt, nil, nil)
}

func storeFilePathsWithDiagnostics(ctx context.Context, result *sync.Map, pt *visuals.ProgressCounter, skipped *[]models.SkippedPath, skippedMu *sync.Mutex) func(string, fs.DirEntry, error) error {
	return func(path string, dirEntry fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			log.WarnWithFuncName(fmt.Sprintf("Stopping file walk at %s due to context cancellation.", path))
			return ctx.Err()
		default:
		}

		if err != nil {
			recordSkippedPath(skipped, skippedMu, path, err)
			reportSkippedWalkEntry(ctx, pt, path, err)
			if dirEntry != nil && dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if dirEntry == nil {
			err := fmt.Errorf("directory entry is unavailable")
			recordSkippedPath(skipped, skippedMu, path, err)
			reportSkippedWalkEntry(ctx, pt, path, err)
			return nil
		}

		if !dirEntry.IsDir() {
			info, err := dirEntry.Info()
			if err != nil {
				recordSkippedPath(skipped, skippedMu, path, err)
				reportSkippedWalkEntry(ctx, pt, path, err)
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

func recordSkippedPath(skipped *[]models.SkippedPath, skippedMu *sync.Mutex, path string, err error) {
	if skipped == nil || skippedMu == nil {
		return
	}
	skippedMu.Lock()
	*skipped = append(*skipped, models.SkippedPath{Path: path, Reason: err.Error()})
	skippedMu.Unlock()
}

func reportSkippedWalkEntry(ctx context.Context, pt *visuals.ProgressCounter, path string, err error) {
	message := fmt.Sprintf("Skipped %s: %v", path, err)
	if errors.Is(err, os.ErrPermission) || os.IsNotExist(err) {
		log.WarnWithFuncName(message)
	} else {
		log.ErrorWithFuncName(message)
	}
	pt.Reporter.LogDetailedStatus(ctx, message)
}
