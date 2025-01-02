package main

import (
	logger "DuDe/common"
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

	mem_arg := "MEMORY_FILE"
	results_arg := "RESULT_FILE"
	source_arg := "SOURCE_DIR"
	target_arg := "TARGET_DIR"

	args := []string{mem_arg, results_arg, source_arg, target_arg}

	log := logger.GetLogger()

	visuals.PrintIntro()

	// myEnv, _ := godotenv.Read()
	loadedArgs := handlers.GetFileArguments(args)
	process.CreateMemoryCSV(loadedArgs[mem_arg])
	hashMemory, err := process.LoadMemoryCSV(loadedArgs[mem_arg])
	logger.PanicAndLog(err)

	// #region paralel
	sourceFiles := make([]models.DuDeFile, 0)

	start := time.Now()

	err = filepath.WalkDir(loadedArgs[source_arg], process.StoreFilePaths(&sourceFiles))
	go visuals.MonitorProgress(len(sourceFiles), progressCh)

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	log.Infof("Started %v", "Paralel")

	availableCPUs := runtime.NumCPU()

	process.StartMemoryUpdateBackground(loadedArgs[mem_arg], memoryChan)
	process.CreateHashes(&sourceFiles, availableCPUs, progressCh, memoryChan, &hashMemory, true)

	elapsed := time.Since(start)
	log.Infof("paralel took: %s for %v files", &elapsed, len(sourceFiles))
	// #endregion paralel

	process.FindDuplicates(&sourceFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)

	err1 := process.SaveResultsAsCSV(flattenedDuplicates, loadedArgs[results_arg])

	if err1 != nil {
		logger.PanicAndLog(err1)
	}
	// visuals.PrintDuplicates(sourceFiles)
}
