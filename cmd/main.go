package main

import (
	logger "DuDe/common"
	handlers "DuDe/internal/handlers"
	process "DuDe/internal/processing"
	st "DuDe/internal/static"
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
		st.GetMemDirTag(),
		st.GetResultDirTag(),
		st.GetSourceDirTag(),
		st.GetTargetDirTag()}

	log := logger.GetLogger()

	visuals.PrintIntro()

	// myEnv, _ := godotenv.Read()
	loadedArgs := handlers.GetFileArguments(args)
	process.CreateMemoryCSV(loadedArgs[st.GetMemDirTag()])
	hashMemory, err := process.LoadMemoryCSV(loadedArgs[st.GetMemDirTag()])
	logger.PanicAndLog(err)

	// #region parallel
	sourceFiles := make([]models.DuDeFile, 0)

	start := time.Now()

	err = filepath.WalkDir(loadedArgs[st.GetSourceDirTag()], process.StoreFilePaths(&sourceFiles))
	go visuals.MonitorProgress(len(sourceFiles), progressCh)

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	availableCPUs := runtime.NumCPU()

	process.StartMemoryUpdateBackground(loadedArgs[st.GetMemDirTag()], memoryChan)
	process.CreateHashes(&sourceFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)

	elapsed := time.Since(start)
	log.Infof("parallel took: %s for %v files", &elapsed, len(sourceFiles))
	// #endregion parallel

	process.FindDuplicates(&sourceFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err1 := process.SaveResultsAsCSV(flattenedDuplicates, loadedArgs[st.GetResultDirTag()])

	if err1 != nil {
		logger.PanicAndLog(err1)
	}
	// visuals.PrintDuplicates(sourceFiles)
}
