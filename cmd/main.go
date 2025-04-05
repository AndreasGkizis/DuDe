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
	timer := time.Now()
	log := logger.Logger

	Args := handlers.LoadArgs()

	visuals.Intro()
	visuals.FirstRun(Args)

	db, err := db.NewDatabase(Args.CacheDir)

	if err != nil {
		logger.ErrorWithFuncName(err.Error())
	}

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

	sourceDirFiles := make([]models.FileHash, 0)
	targetDirFiles := make([]models.FileHash, 0)

	go process.WalkDir(Args.SourceDir, &sourceDirFiles, rt)

	if Args.DualFolderModeEnabled {
		go process.WalkDir(Args.TargetDir, &targetDirFiles, rt)
	}
	rt.Wait()

	pt := visuals.NewProgressTracker("Hashing\t\t")
	pt.Start(50)

	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	process.CreateHashes(&sourceDirFiles, Args.Cpus, pt, mm, &hashMemory, &failedCounter)

	if Args.DualFolderModeEnabled {

		process.CreateHashes(&targetDirFiles, Args.Cpus, pt, mm, &hashMemory, &failedCounter)
	}
	mm.Wait()

	if Args.DualFolderModeEnabled {
		process.FindDuplicates(&sourceDirFiles, &targetDirFiles)
	} else {
		process.FindDuplicates(&sourceDirFiles)
	}

	duplicates := process.GetDuplicates(&sourceDirFiles)

	dupsFound := len(duplicates) != 0
	if dupsFound {
		timer1 := time.Now()
		compareTracker := visuals.NewProgressTracker("Comparing\t")
		compareTracker.Start(50)

		duplicates, err = process.EnsureDuplicates(duplicates, compareTracker, Args.Cpus)
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

	visuals.Outro()
}
