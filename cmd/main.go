package main

import (
	common "DuDe/internal/common"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	models "DuDe/internal/models"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	"time"

	"path/filepath"
	"runtime"
)

func main() {
	timer := time.Now()
	log := common.Logger

	availableCPUs := runtime.NumCPU()
	Args := handlers.LoadArgs()

	visuals.Intro()
	visuals.FirstRun(Args)

	db, err := db.NewDatabase(Args.CacheDir)

	if err != nil {
		common.Logger.Panicf(err.Error())
	}

	pt := visuals.NewProgressTracker()
	pt.Start(50)
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

	duplicates := process.GetDuplicates(&sourceDirFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err = process.SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memorychan", failedCounter)

	if err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
}
