package services

import (
	com "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HashError identifies a file that could not be inspected during hashing.
type HashError struct {
	Path string
	Err  error
}

func (e HashError) Error() string {
	return fmt.Sprintf("hash %q: %v", e.Path, e.Err)
}

func (e HashError) Unwrap() error {
	return e.Err
}

func runWorkers(ctx context.Context, sourceFiles *sync.Map, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash, submitHashError func(error) bool) error {
	defer pt.Complete()

	if maxWorkers < 1 {
		mm.SenderFinished()
		return fmt.Errorf("max workers must be at least 1")
	}

	numFilesToHash := com.LenSyncMap(sourceFiles)
	if numFilesToHash == 0 {
		mm.SenderFinished()
		return nil // Nothing to hash
	}

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started hashing %d files with %d workers", groupID, numFilesToHash, maxWorkers))
	pt.AddTotal(int64(numFilesToHash))

	var wg sync.WaitGroup

	sem := make(chan struct{}, maxWorkers) // Define semaphore with buffer size

	sourceFiles.Range(func(key, value interface{}) bool {

		select {
		case <-ctx.Done():
			log.DebugWithFuncName(fmt.Sprintf("Group %d CreateHashes stopped spawning workers due to context cancellation.", groupID))
			return false // Stop sourceFiles.Range loop
		default:
			// Continue spawning worker
		}

		wg.Add(1)
		go func(path string, val models.FileHash) {
			defer wg.Done()
			var hash string
			var err error

			currentFilePath := key.(string)

			// Acquire a slot
			select {
			case <-ctx.Done():
				// Context canceled while waiting for the semaphore
				log.DebugWithFuncName(fmt.Sprintf("Worker skipped file. context canceled while waiting for semaphore. | filepath: %s", currentFilePath))
				return // Exit the worker goroutine
			case sem <- struct{}{}:
				// Slot acquired, proceed
			}
			defer func() { <-sem }() // Release the slots
			if ctx.Err() != nil {
				log.DebugWithFuncName(fmt.Sprintf("Worker skipped file. context canceled immediately after semaphore acquisition. | filepath: %s", currentFilePath))
				return
			}

			currentFileDiskStats, err := os.Stat(val.FilePath)
			if err != nil {
				fatal := submitHashError(HashError{Path: val.FilePath, Err: err})
				sourceFiles.Delete(val.FilePath)
				pt.DecrementFromTotal() // remove for progress bar
				if fatal {
					return
				}
				return // stop this iteration
			}

			currentFileDiskSize := currentFileDiskStats.Size()
			currentFileDiskModTime := currentFileDiskStats.ModTime().Format(com.TimeFrmt)

			memoryOfFile, memoryExists := (*memory)[currentFilePath]

			fileHasChangedOnDisk := memoryOfFile.FileSize != currentFileDiskSize || memoryOfFile.ModTime != currentFileDiskModTime

			fileNeedsReHashing := !memoryExists || fileHasChangedOnDisk

			if fileNeedsReHashing {
				hash, err = calculateMD5Hash(ctx, val)
				if errors.Is(err, context.Canceled) {
					log.DebugWithFuncName(fmt.Sprintf("Hashing stopped due to context cancellation. | filepath: %s", currentFilePath))
					return // Stop this iteration/worker
				}
				if err != nil {
					fatal := submitHashError(HashError{Path: val.FilePath, Err: err})
					sourceFiles.Delete(val.FilePath)
					pt.DecrementFromTotal() // remove for progress bar
					if fatal {
						return
					}
					return // stop this iteration
				}

				newMem := models.FileHash{
					FileName: filepath.Base(path),
					FilePath: path,
					Hash:     hash,
					FileSize: currentFileDiskSize,
					ModTime:  currentFileDiskModTime,
				}

				sourceFiles.Store(path, newMem)
				mm.Push(newMem)

			} else {
				sourceFiles.Store(path, memoryOfFile)
			}

			// safeResend(mm.Channel, newMem, 500*time.Microsecond), something to never miss a new memory?

			pt.Increment()
		}(key.(string), value.(models.FileHash))
		return true
	})

	wg.Wait()
	mm.SenderFinished()
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished hashing %d files with %d workers", groupID, int64(numFilesToHash), maxWorkers))

	close(sem)
	// Check if the overall context was canceled or nil if not
	return ctx.Err()
}

// CreateHashes hashes files and routes path-specific failures through the execution sink.
func CreateHashes(ctx context.Context, sourceFiles *sync.Map, maxWorkers int, pt *visuals.ProgressTracker, mm *MemoryManager, memory *map[string]models.FileHash, sink *DiagnosticSink, handler DiagnosticHandler, scope models.DiagnosticScope) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var firstFatal error
	var submitMu sync.Mutex
	submit := func(hashErr error) bool {
		var diagnostic models.Diagnostic
		var typed HashError
		if errors.As(hashErr, &typed) {
			diagnostic = models.Diagnostic{Path: typed.Path, Reason: typed.Err.Error(), Phase: models.DiagnosticPhaseHashing, Scope: scope}
		} else {
			diagnostic = models.Diagnostic{Reason: hashErr.Error(), Phase: models.DiagnosticPhaseHashing, Scope: scope}
		}

		submitMu.Lock()
		defer submitMu.Unlock()
		event := DiagnosticEvent{Diagnostic: diagnostic, Err: hashErr, Severity: DiagnosticSeverityWarning}
		disposition := handler(event)
		sink.Submit(event, disposition)
		if disposition == ErrorFatal {
			if firstFatal == nil {
				firstFatal = hashErr
			}
			cancel()
			return true
		}
		return false
	}

	err := runWorkers(ctx, sourceFiles, maxWorkers, pt, mm, memory, submit)
	if firstFatal != nil {
		return firstFatal
	}
	return err
}

func EnsureDuplicates(ctx context.Context, input *sync.Map, pt *visuals.ProgressTracker, maxWorkers int) error {
	defer pt.Complete()

	num := 0

	if ctx.Err() != nil {
		log.DebugWithFuncName("EnsureDuplicates skipped due to context cancellation.")
		return ctx.Err()
	}
	if maxWorkers < 1 {
		return fmt.Errorf("max workers must be at least 1")
	}

	// Count every file comparison so progress has an accurate total.
	input.Range(func(_, value any) bool {
		num += len(value.(models.FileHash).DuplicatesFound)
		return true
	})

	if num == 0 {
		log.WarnWithFuncName("No duplicates to ensure")
	}

	pt.AddTotal(int64(num))

	// Limit concurrent groups and collect errors safely from all workers.
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)
	var errorMu sync.Mutex
	comparisonErrors := make([]error, 0)
	recordError := func(err error) {
		errorMu.Lock()
		comparisonErrors = append(comparisonErrors, err)
		errorMu.Unlock()
	}

	// Each map entry is one hash group with a primary file and its matches.
	input.Range(func(itemHash, item any) bool {
		select {
		case <-ctx.Done():
			log.DebugWithFuncName("stopped spawning workers due to context cancellation.")
			return false
		default:
		}

		wg.Add(1)
		go func(itemHash string, item models.FileHash) {
			defer wg.Done()

			// Wait until this group has an available worker slot.
			select {
			case <-ctx.Done():
				log.DebugWithFuncName(fmt.Sprintf("Worker for hash %s skipped: context canceled while waiting for semaphore.", itemHash))
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			// Cancellation may happen while waiting for the worker slot.
			if ctx.Err() != nil {
				log.DebugWithFuncName(fmt.Sprintf("Worker for hash %s skipped: context canceled immediately after semaphore acquisition.", itemHash))
				return
			}

			if len(item.DuplicatesFound) == 0 {
				return
			}

			// Open the primary file once and compare every candidate against it.
			mainFile, err := os.Open(item.FilePath)
			if err != nil {
				recordError(fmt.Errorf("open primary file %q: %w", item.FilePath, err))
				input.Delete(itemHash)
				for range item.DuplicatesFound {
					pt.Increment()
				}
				return
			}

			// Build a fresh list containing only byte-for-byte matches.
			confirmedDuplicates := make([]models.FileHash, 0, len(item.DuplicatesFound))

			for duplicateIndex, duplicate := range item.DuplicatesFound {
				// Stop comparing this group when the execution is cancelled.
				select {
				case <-ctx.Done():
					log.WarnWithFuncName(fmt.Sprintf("Worker for hash %s stopped mid-comparison loop due to cancellation.", itemHash))
					for range item.DuplicatesFound[duplicateIndex:] {
						pt.Increment()
					}
					if err := mainFile.Close(); err != nil {
						recordError(fmt.Errorf("close primary file %q: %w", item.FilePath, err))
					}
					return
				default:
				}

				// Comparison errors are recorded and never treated as matches.
				equal, err := FilesMatch(ctx, mainFile, item, duplicate, true)
				pt.Increment()

				if err != nil {
					recordError(fmt.Errorf("compare %q with %q: %w", item.FilePath, duplicate.FilePath, err))
					continue
				}

				if equal {
					confirmedDuplicates = append(confirmedDuplicates, duplicate)
				}

				// Rewind the primary file before comparing the next candidate.
				if _, err := mainFile.Seek(0, io.SeekStart); err != nil {
					recordError(fmt.Errorf("reset primary file %q: %w", item.FilePath, err))
					for range item.DuplicatesFound[duplicateIndex+1:] {
						pt.Increment()
					}
					break
				}
			}

			// Closing errors also make the verification phase fail visibly.
			if err := mainFile.Close(); err != nil {
				recordError(fmt.Errorf("close primary file %q: %w", item.FilePath, err))
			}

			if len(confirmedDuplicates) == 0 {
				input.Delete(itemHash)
				return
			}

			// Replace the original group with its verified matches.
			item.DuplicatesFound = confirmedDuplicates
			input.Store(itemHash, item)
		}(itemHash.(string), item.(models.FileHash))
		return true
	})

	// Do not return until every comparison worker has finished.
	wg.Wait()
	if ctx.Err() != nil {
		comparisonErrors = append(comparisonErrors, ctx.Err())
	}
	return errors.Join(comparisonErrors...)
}

func calculateMD5Hash(ctx context.Context, file models.FileHash) (string, error) {

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	hasherMD5 := md5.New()

	f, err := os.Open(file.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			log.WarnWithFuncName(fmt.Sprintf("Skipping file: %s, reason: %s", file.FilePath, err.Error()))
			return "", err
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.ErrorWithFuncName(err.Error())
		}
	}()

	// 💡 Performance Note: For cancellation during long reads,
	// you would need a custom Reader that checks ctx.Done() periodically.
	// For now, we assume the open/close is the main blocking point.
	if _, err := io.Copy(hasherMD5, f); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	} // TODO: add blob suffix for uniquness
	return fmt.Sprintf("%x", hasherMD5.Sum(nil)), nil
}

func FindDuplicatesInMap(ctx context.Context, fileHashes *sync.Map, tracker *visuals.ProgressTracker) {
	defer tracker.Complete()

	timer := time.Now()
	initialCount := com.LenSyncMap(fileHashes)

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started for source folder with %d files", groupID, initialCount))

	hashCounts := make(map[string]int)
	hashPaths := make(map[string][]models.FileHash)

	fileHashes.Range(func(_, value any) bool {

		select {
		case <-ctx.Done():
			log.DebugWithFuncName(fmt.Sprintf("Group %d stopped grouping by hash due to context cancellation.", groupID))
			return false // Stop fileHashes.Range loop
		default:
			// Continue
		}

		hash := value.(models.FileHash).Hash

		hashCounts[hash]++
		hashPaths[hash] = append(hashPaths[hash], value.(models.FileHash))
		return true
	})
	if ctx.Err() != nil {
		return // Exit the function entirely
	}

	totalGroups := len(hashPaths)
	tracker.AddTotal(int64(totalGroups))

	fileHashes.Clear()

	for hash, files := range hashPaths {

		select {
		case <-ctx.Done():
			log.DebugWithFuncName(fmt.Sprintf("Group %d stopped processing hash groups due to context cancellation.", groupID))
			return // Exit the function entirely
		default:
			// Continue
		}

		if len(files) == 1 {
			delete(hashPaths, hash)
			tracker.Increment()
		} else {
			file := files[0] // smallest name?
			dups := []models.FileHash{}
			for i := 1; i < len(files); i++ {
				dups = append(dups, files[i])
			}
			file.DuplicatesFound = dups
			fileHashes.Store(file.Hash, file)
			tracker.Increment()
		}
	}

	log.InfoWithFuncName(fmt.Sprintf("Group %d finished and, took : %s .source folder with %d files", groupID, time.Since(timer), initialCount))
}

func GetFlattened(input *sync.Map) []models.ResultEntry {
	result := make([]models.ResultEntry, 0)

	separatorEntry := models.ResultEntry{
		Filename:          com.ResultsFileSeperator,
		FullPath:          com.ResultsFileSeperator,
		DuplicateFilename: com.ResultsFileSeperator,
		DuplicateFullPath: com.ResultsFileSeperator}

	input.Range(func(key, value any) bool {
		val := value.(models.FileHash)
		for _, dup := range val.DuplicatesFound {
			a := models.ResultEntry{
				Filename:          val.FileName,
				FullPath:          val.FilePath,
				DuplicateFilename: dup.FileName,
				DuplicateFullPath: dup.FilePath,
			}
			result = append(result, a)
		}
		result = append(result, separatorEntry)

		return true
	})
	return result
}
