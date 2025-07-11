package main

import (
	"DuDe/internal/common"
	logger "DuDe/internal/common/logger"
	db "DuDe/internal/db"
	handlers "DuDe/internal/handlers"
	process "DuDe/internal/processing"
	visuals "DuDe/internal/visuals"
	"fmt"
	"sync"
	"time"
)

func main() {
	// debug.SetMemoryLimit(2750 * 1 << 20) // 2750 MB

	timer := time.Now()
	log := logger.Logger

	Args := handlers.LoadArgs()

	visuals.Intro()

	db, err := db.NewDatabase(Args.CacheDir)

	if err != nil {
		logger.ErrorWithFuncName(err.Error())
	}

	errChan := make(chan error, 100)
	go func() {
		for err := range errChan {
			logger.WarnWithFuncName(err.Error())
		}
	}()

	var senderGroups int32
	if Args.DualFolderModeEnabled {
		senderGroups = 2
	} else {
		senderGroups = 1
	}

	failedCounter := 0
	mm := process.NewMemoryManager(db, Args.BufSize, 1)
	mm.Start()
	rt := visuals.NewProgressCounter("Reading\t\t", int(senderGroups))
	rt.Start()
	// ^^^ slightly hacky and dump but works for now.

	hashMemory := mm.LoadMemory()

	var syncSourceDirFileMap sync.Map

	go process.WalkDir(Args.SourceDir, &syncSourceDirFileMap, rt)

	if Args.DualFolderModeEnabled {

		go process.WalkDir(Args.TargetDir, &syncSourceDirFileMap, rt)
	}
	rt.Wait()
	len := common.LenSyncMap(&syncSourceDirFileMap)
	if len == 0 {
		visuals.EmptyDir("asd")
	}

	pt := visuals.NewProgressTracker("Hashing\t\t")
	pt.Start(50)

	// bla := common.LenSyncMap(&syncSourceDirFileMap)
	// bla1 := common.ConvertSyncMapToMap(&syncSourceDirFileMap)
	// log.Debug(bla1)

	err = process.CreateHashes(&syncSourceDirFileMap, Args.CPUs, pt, mm, &hashMemory, &failedCounter, errChan)
	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Hashing directory: %v", err))
	}

	pt.Wait()
	mm.Wait()
	close(errChan)

	findTracker := visuals.NewProgressTracker("Finding\t\t")
	findTracker.Start(50)

	process.FindDuplicatesInMap(&syncSourceDirFileMap, findTracker)

	findTracker.Wait()
	length := common.LenSyncMap(&syncSourceDirFileMap)

	logger.InfoWithFuncName(fmt.Sprintf("found %v duplicates", length))
	if length != 0 {
		timer1 := time.Now()

		if Args.ParanoidMode {
			compareTracker := visuals.NewProgressTracker("Comparing\t")
			compareTracker.Start(50)

			process.EnsureDuplicates(&syncSourceDirFileMap, compareTracker, Args.CPUs)

			compareTracker.Wait()
		}

		flattenedDuplicates := process.GetFlattened(&syncSourceDirFileMap)
		err = process.SaveResultsAsCSV(flattenedDuplicates, Args.ResultsDir)

		if err != nil {
			log.Fatalf("Error saving result: %v", err)
		}

		log.Infof("Took: %s to look through bytes", time.Since(timer1))
	} else {
		visuals.NoDuplicatesFound()
		log.Info("No duplicates were found")
	}

	log.Infof("Took: %s for buffer size %d", time.Since(timer), Args.BufSize)
	log.Infof("Failed %d times to send to memoryChan", failedCounter)

	visuals.Outro()
}
