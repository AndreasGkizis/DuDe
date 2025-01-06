package main

import (
	common "DuDe/common"
	handlers "DuDe/internal/handlers"
	process "DuDe/internal/processing"
	"DuDe/internal/visuals"
	"DuDe/models"
	"path/filepath"
	"runtime"
	"time"
)

func init() {
	process.CreateArgsFile()
}

func main() {
	progressCh := make(chan int)
	memoryChan := make(chan models.FileHash)

	args := []string{
		common.ArgFilename_cacheDir,
		common.ArgFilename_resDir,
		common.ArgFilename_sourceDir,
		common.ArgFilename_targetDir}

	log := common.GetLogger()

	visuals.PrintIntro()

	loadedArgs := handlers.GetFileArguments(args)
	//override file args with cli
	loadedArgs = handlers.GetCLIArgs(loadedArgs)

	process.CreateMemoryCSV(common.ArgFilename)
	hashMemory, err := process.LoadMemoryCSV(loadedArgs[common.ArgFilename_cacheDir])
	common.PanicAndLog(err)

	// #region parallel
	sourceFiles := make([]models.DuDeFile, 0)

	start := time.Now()

	err = filepath.WalkDir(loadedArgs[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceFiles))
	go visuals.MonitorProgress(len(sourceFiles), progressCh)

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	availableCPUs := runtime.NumCPU()

	process.StartMemoryUpdateBackground(loadedArgs[common.ArgFilename_cacheDir], memoryChan)

	process.CreateHashes(&sourceFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)

	elapsed := time.Since(start)
	log.Infof("parallel took: %s for %v files", &elapsed, len(sourceFiles))
	// #endregion parallel

	process.FindDuplicates(&sourceFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err1 := process.SaveResultsAsCSV(flattenedDuplicates, loadedArgs[common.ArgFilename_resDir])

	if err1 != nil {
		common.PanicAndLog(err1)
	}
	// visuals.PrintDuplicates(sourceFiles)
}
