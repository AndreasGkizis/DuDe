package main

import (
	"DuDe/common"
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

	progressCh := make(chan int)
	memoryChan := make(chan models.FileHash, 100)

	args := []string{
		common.ArgFilename_cacheDir,
		common.ArgFilename_resDir,
		common.ArgFilename_sourceDir,
		common.ArgFilename_targetDir}

	log := common.Logger

	visuals.PrintIntro()

	// TODO: refactor into single method

	loadedArgs := handlers.GetFileArguments(args)
	//override file args with cli
	loadedArgs = handlers.GetCLIArgs(loadedArgs)

	db, err := db.NewDatabase(loadedArgs[common.ArgFilename_cacheDir], loadedArgs[common.DbgFlagName] == common.DbgFlagActiveValue)
	common.PanicAndLog(err)
	hashMemory := process.LoadMemory(db)

	// #region parallel
	sourceFiles := make([]models.DuDeFile, 0)
	targetFiles := make([]models.DuDeFile, 0)

	err = filepath.WalkDir(loadedArgs[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceFiles))

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	// err = filepath.WalkDir(loadedArgs[common.ArgFilename_targetDir], process.StoreFilePaths(&targetFiles))

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	go visuals.MonitorProgress(len(sourceFiles)+len(targetFiles), progressCh)

	availableCPUs := runtime.NumCPU()
	// process.StartMemoryUpdateBackgroundProcess(loadedArgs[common.ArgFilename_cacheDir], memoryChan)

	process.CreateHashes(&sourceFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)
	// process.CreateHashes(&targetFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)

	close(memoryChan)
	process.Updatewitwg(db, memoryChan)

	// #endregion parallel

	process.FindDuplicates(&sourceFiles, &targetFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err1 := process.SaveResultsAsCSV(flattenedDuplicates, loadedArgs[common.ArgFilename_resDir])

	if err1 != nil {
		common.PanicAndLog(err1)
	}
	// visuals.PrintDuplicates(sourceFiles)

}
