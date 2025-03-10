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

	db, err := db.NewDatabase(Args[common.ArgFilename_cacheDir])

	if err != nil {
		common.Logger.Panicf(err.Error())
	}

	pt := visuals.NewProgressTracker()
	pt.Start(50)
	bufsize := 100
	failedCounter := 0
	mm := process.NewMemoryManager(db, bufsize)
	mm.Start()

	hashMemory := mm.LoadMemory()

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)

	if Args[common.ArgFilename_Mode] == common.ModeDualFolder {
		err = filepath.WalkDir(Args[common.ArgFilename_targetDir], process.StoreFilePaths(&targetDirFiles))

		if err != nil {
			log.Fatalf("Error walking directory: %v", err)
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	}

	err = filepath.WalkDir(Args[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceDirFiles))

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}
	log.Infof("Number of files: %d ", len(sourceDirFiles))

	process.CreateHashes(&sourceDirFiles, availableCPUs, pt, mm, &hashMemory, &failedCounter)
	mm.Wait()

	if Args[common.ArgFilename_Mode] == common.ModeDualFolder {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	duplicates := process.GetDuplicates(&sourceDirFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err = process.SaveResultsAsCSV(flattenedDuplicates, Args[common.ArgFilename_resDir])

	log.Infof("Took: %s for buffer size %d", time.Since(timer), bufsize)
	log.Infof("Failed %d times to send to memorychan", failedCounter)

	if err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
}
