package main

import (
	common "DuDe/common"
	db "DuDe/internal/db"
	"DuDe/internal/handlers"
	process "DuDe/internal/processing"
	"DuDe/internal/visuals"
	"DuDe/models"
	"path/filepath"
	"runtime"
)

func init() {
	process.CreateArgsFile()
}

func main() {

	progressCh := make(chan int, 100)
	memoryChan := make(chan models.FileHash, 100) // in theory the files get hashed much slower than they get saved, so this would remain empty for a most time. needs investigating

	log := common.Logger
	availableCPUs := runtime.NumCPU()
	Args := handlers.LoadArgs()

	dualFolderMode := Args[common.ArgFilename_targetDir] != common.Def

	db, err := db.NewDatabase(Args[common.ArgFilename_cacheDir], Args[common.DbgFlagName] == common.DbgFlagActiveValue)
	common.PanicAndLog(err)

	hashMemory := process.LoadMemory(db)

	sourceDirFiles := make([]models.DuDeFile, 0)
	targetDirFiles := make([]models.DuDeFile, 0)

	if dualFolderMode {
		err = filepath.WalkDir(Args[common.ArgFilename_targetDir], process.StoreFilePaths(&targetDirFiles))

		if err != nil {
			log.Fatalf("Error walking directory: %v", err)
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)
	}

	err = filepath.WalkDir(Args[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceDirFiles))

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	go visuals.MonitorProgress(len(sourceDirFiles)+len(targetDirFiles), progressCh)

	process.CreateHashes(&sourceDirFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)

	close(memoryChan)
	process.Updatewitwg(db, memoryChan)

	if dualFolderMode {
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
