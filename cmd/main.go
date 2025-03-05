package main

import (
	common "DuDe/common"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	models "DuDe/models"

	"path/filepath"
	"runtime"
)

func init() {
	process.CreateArgsFile()
}

func main() {
	log := common.Logger

	availableCPUs := runtime.NumCPU()
	Args := handlers.LoadArgs()

	visuals.PrintIntro()

	// progressCh := make(chan int, 100)
	memoryChan := make(chan models.FileHash, 1000)

	db, err := db.NewDatabase(Args[common.ArgFilename_cacheDir])
	common.PanicAndLog(err)

	pt := visuals.NewProgressTracker()
	pt.Start(50)

	mt := process.NewMemoryTracker(db)
	mt.Start(memoryChan)

	hashMemory := process.LoadMemory(db)

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)

	if Args[common.ArgFilename_Mode] == common.ModeDualFolder {
		err = filepath.WalkDir(Args[common.ArgFilename_targetDir], process.StoreFilePaths(&targetDirFiles))

		if err != nil {
			log.Fatalf("Error walking directory: %v", err)
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, pt, memoryChan, &hashMemory)
	}

	err = filepath.WalkDir(Args[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceDirFiles))

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	process.CreateHashes(&sourceDirFiles, availableCPUs, pt, memoryChan, &hashMemory)

	if Args[common.ArgFilename_Mode] == common.ModeDualFolder {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	duplicates := process.GetDuplicates(&sourceDirFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err = process.SaveResultsAsCSV(flattenedDuplicates, Args[common.ArgFilename_resDir])

	if err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
}
