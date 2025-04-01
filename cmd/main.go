package main

import (
	logger "DuDe/internal/common/logger"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	models "DuDe/internal/models"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	"fmt"
	"time"

	"path/filepath"
	"runtime"
)

func main() {
	timer := time.Now()
	log := logger.Logger

	availableCPUs := runtime.NumCPU()
	Args := handlers.LoadArgs()

	visuals.Intro()
	visuals.FirstRun(Args)

	db, err := db.NewDatabase(Args.CacheDir)

	if err != nil {
		logger.ErrorWithFuncName(err.Error())
	}

	pt := visuals.NewProgressTracker("Hashing\t\t")
	pt.Start(50)

	failedCounter := 0
	mm := process.NewMemoryManager(db, Args.BufSize)
	mm.Start()

	if Args.DualFolderModeEnabled {
		mm.TotalSenders(2)
	} else {
		mm.TotalSenders(1)
	}

	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)
	// var skipped []string

	if Args.DualFolderModeEnabled {

		err = filepath.WalkDir(Args.TargetDir, process.StoreFilePaths(&targetDirFiles))
		// err = filepath.WalkDir(Args.TargetDir, process.StoreFilePaths(&targetDirFiles, &skipped))

		if err != nil {
			logger.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	}
	err = filepath.WalkDir(Args.SourceDir, process.StoreFilePaths(&sourceDirFiles))
	//  err = filepath.WalkDir(Args.SourceDir, process.StoreFilePaths(&sourceDirFiles, &skipped))

	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	process.CreateHashes(&sourceDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	mm.Wait()

	if Args.DualFolderModeEnabled {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	duplicates := process.GetDuplicates(&sourceDirFiles)

	dupsFound := len(duplicates) != 0
	if dupsFound {
		timer1 := time.Now()
		compareTracker := visuals.NewProgressTracker("Comparing\t")
		compareTracker.Start(50)

		duplicates, err = process.EnsureDuplicates(duplicates, compareTracker, Args.Cpus)
		if err != nil {
			log.Fatalf("Error Comparing results: %v", err)
		}
		flattenedDuplicates := process.GetFlattened(&duplicates)
		err = process.SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)

		if err != nil {
			log.Fatalf("Error saving result: %v", err)
		}

		compareTracker.Wait()
		log.Infof("Took: %s to look through bytes", time.Since(timer1))
	} else {
		visuals.NoDuplicatesFound()
		log.Info("No duplicates were found")
	}

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memorychan", failedCounter)

	pt.Wait()

	visuals.Outro()
}
