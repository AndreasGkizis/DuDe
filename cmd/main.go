package main

import (
	common "DuDe/common"
	db "DuDe/internal/db"
	"DuDe/internal/handlers"
	process "DuDe/internal/processing"
	"DuDe/internal/visuals"
	"DuDe/models"
	"fmt"
	_ "net/http"       //for profiling
	_ "net/http/pprof" //for profiling
	"path/filepath"
	"runtime"
	"time"
)

func init() {
	process.CreateArgsFile()
}

func main() {
	var maxAlloc uint64

	// Function to update maxAlloc
	updateMaxAlloc := func() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > maxAlloc {
			maxAlloc = m.Alloc
		}
	}

	// Defer a function to print the maxAlloc at the end
	defer func() {
		fmt.Printf("\nMaximum memory allocated: %d bytes (%.2f MB)\n", maxAlloc, float64(maxAlloc)/(1024*1024))
	}()

	// http.ListenAndServe("localhost:8080", nil) // for profiling
	start := time.Now()
	log := common.Logger

	availableCPUs := runtime.NumCPU()
	Args := handlers.LoadArgs()

	visuals.PrintIntro()

	progressCh := make(chan int, 100)
	memoryChan := make(chan models.FileHash, 1000) // in theory the files get hashed much slower than they get saved, so this would remain empty for a most time. needs investigating

	db, err := db.NewDatabase(Args[common.ArgFilename_cacheDir])
	common.PanicAndLog(err)
	updateMaxAlloc()
	pt := visuals.NewProgressTracker()
	pt.Start(50, progressCh)

	mt := process.NewMemoryTracker(db)
	mt.Start(memoryChan)

	dualFolderMode := Args[common.ArgFilename_targetDir] != common.Def
	hashMemory := process.LoadMemory(db)

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)

	if dualFolderMode {
		err = filepath.WalkDir(Args[common.ArgFilename_targetDir], process.StoreFilePaths(&targetDirFiles))

		if err != nil {
			log.Fatalf("Error walking directory: %v", err)
		}

		process.CreateHashes(&targetDirFiles, availableCPUs, pt, progressCh, memoryChan, &hashMemory)
	}

	err = filepath.WalkDir(Args[common.ArgFilename_sourceDir], process.StoreFilePaths(&sourceDirFiles))

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	process.CreateHashes(&sourceDirFiles, availableCPUs, pt, progressCh, memoryChan, &hashMemory)
	updateMaxAlloc()

	if dualFolderMode {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	duplicates := process.GetDuplicates(&sourceDirFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)
	updateMaxAlloc()
	err = process.SaveResultsAsCSV(flattenedDuplicates, Args[common.ArgFilename_resDir])
	fmt.Printf("execution took : %s", time.Since(start))
	if err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
}
