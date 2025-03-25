package main

import (
	"DuDe/internal/common/logger"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	models "DuDe/internal/models"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	"sync"
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
		log.Panicf(err.Error())
	}

	pt := visuals.NewProgressTracker("Hashing\t\t")
	pt.Start(50)

	var taskWG sync.WaitGroup
	taskWG.Wait()

	failedCounter := 0
	mm := process.NewMemoryManager(db, Args.BufSize)
	mm.Start()

	hashMemory := mm.LoadMemory()

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)

	if Args.IsDualFolderMode() {
		err = filepath.WalkDir(Args.TargetDir, process.StoreFilePaths(&targetDirFiles))

		if err != nil {
			log.Fatalf("Error walking directory: %v", err)
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	}

	err = filepath.WalkDir(Args.SourceDir, process.StoreFilePaths(&sourceDirFiles))

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}
	log.Infof("Number of files: %d ", len(sourceDirFiles))

	process.CreateHashes(&sourceDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	mm.Wait()

	if Args.IsDualFolderMode() {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	compareTracker := visuals.NewProgressTracker("Comparing\t")
	compareTracker.Start(50)

	duplicates := process.GetDuplicates(&sourceDirFiles)
	timer1 := time.Now()
	duplicates, err = process.EnsureDuplicates(duplicates, compareTracker, Args.Cpus)
	if err != nil {
		log.Fatalf("Error Comparing results: %v", err)
	}

	log.Infof("Took: %s to look through bytes", time.Since(timer1))

	flattenedDuplicates := process.GetFlattened(&duplicates)

	err = process.SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memorychan", failedCounter)

	if err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
	pt.Wait()
	compareTracker.Wait()
}
