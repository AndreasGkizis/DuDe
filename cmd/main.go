package main

import (
	logger "DuDe/common"
	process "DuDe/internal/processing"
	"DuDe/internal/visuals"
	"DuDe/models"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// loads values from .env into the system
	log := logger.GetLogger()

	if err := godotenv.Load(); err != nil {
		log.Warnln("No .env file found")
	}
}

func main() {
	log := logger.GetLogger()
	visuals.PrintIntro()
	// cliArgs := handlers.GetCLIArgs()
	myEnv, _ := godotenv.Read()
	// v := handlers.GetFileArguments()

	sem := make(chan struct{}, 8) // semaphore which allows 8 workers at the same time

	// go visuals.MonitorGoroutines(sem)

	// #region paralel
	sourceFiles := make([]models.DuDeFile, 0)
	timer := time.Now()

	filepath.WalkDir(myEnv["SOURCE"], process.StoreFilePaths(&sourceFiles))

	log.Infof("Started %v", "Paralel")

	var wg sync.WaitGroup
	mutex := sync.Mutex{}

	for i := range sourceFiles {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// using struct{}{} since it allocates nothing , it is a pure signal
			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			hash, _ := process.CalculateMD5Hash(sourceFiles[index])
			name := process.GetFileName(sourceFiles[index].FullPath)
			mutex.Lock()
			sourceFiles[index].Hash = hash
			sourceFiles[index].Filename = name
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	close(sem)
	elapsed1 := time.Since(timer)
	log.Infof("paralel took: %s for %v files", &elapsed1, len(sourceFiles))
	// #endregion paralel

	process.FindDuplicates(&sourceFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)
	process.SaveAsCSV(flattenedDuplicates, "/home/andreas/Code/GoLang/DuDe/cmd/results.csv")

	// visuals.PrintDuplicates(sourceFiles)
}
