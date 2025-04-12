package main

import (
	logger "DuDe/internal/common/logger"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	models "DuDe/internal/models"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	"fmt"
	"time"
)

func main() {
	// debug.SetMemoryLimit(2750 * 1 << 20) // 2750 MB

	timer := time.Now()
	log := logger.Logger

	Args := handlers.LoadArgs()

	visuals.Intro()
	// visuals.FirstRun(Args)

	db, err := db.NewDatabase(Args.CacheDir)

	if err != nil {
		logger.ErrorWithFuncName(err.Error())
	}

	errChan := make(chan error, 100)

	failedCounter := 0
	mm := process.NewMemoryManager(db, Args.BufSize)
	mm.Start()
	rt := visuals.NewProgressCounter("Reading\t\t")
	rt.Start()

	var senderGroups int32
	if Args.DualFolderModeEnabled {
		senderGroups = 2
	} else {
		senderGroups = 1
	}

	mm.TotalSenders(senderGroups)
	rt.TotalSenders(senderGroups)
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	sourceDirFilesmap := make(map[string]models.FileHash)
	targetDirFilesmap := make(map[string]models.FileHash)

	go process.WalkDir(Args.SourceDir, &sourceDirFilesmap, rt)

	if Args.DualFolderModeEnabled {
		go process.WalkDir(Args.TargetDir, &targetDirFilesmap, rt)
	}
	rt.Wait()

	pt := visuals.NewProgressTracker("Hashing\t\t")
	pt.Start(50)

	err = process.CreateHashes(&sourceDirFilesmap, Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
	}

	if Args.DualFolderModeEnabled {

		err = process.CreateHashes(&targetDirFilesmap, Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
		if err != nil {
			logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
		}
	}
	mm.Wait()

	if Args.DualFolderModeEnabled {
		process.FindDuplicatesBetweenMaps(&sourceDirFilesmap, &targetDirFilesmap)
	} else {
		process.FindDuplicatesInMap(&sourceDirFilesmap)
	}

	duplicates := process.GetDuplicates(&sourceDirFilesmap)

	if len(duplicates) != 0 {
		timer1 := time.Now()
		compareTracker := visuals.NewProgressTracker("Comparing\t")
		compareTracker.Start(50)

		duplicates, err = process.EnsureDuplicates(duplicates, compareTracker, Args.CPUs)
		if err != nil {
			log.Fatalf("Error Comparing results: %v", err)
		}
		flattenedDuplicates := process.GetFlattened(&duplicates)
		err = process.SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)

		if err != nil {
			log.Fatalf("Error saving result: %v", err)
		}

		compareTracker.Wait()
		log.Infof("Took: %s to look through bytes", time.Since(timer1))
	} else {
		visuals.NoDuplicatesFound()
		log.Info("No duplicates were found")
	}

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memorychan", failedCounter)

	pt.Wait()

	go func() {
		for err := range errChan {
			logger.WarnWithFuncName(err.Error())
		}
	}()
	close(errChan)

	visuals.Outro()
}
