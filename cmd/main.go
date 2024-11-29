package main

import (
	logger "DuDe/common"
	process "DuDe/internal/processing"
	"DuDe/internal/visuals"
	"DuDe/models"
	"path/filepath"
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

	// go visuals.MonitorGoroutines(sem)

	// #region paralel
	sourceFiles := make([]models.DuDeFile, 0)
	timer := time.Now()
	err := filepath.WalkDir(myEnv["SOURCE"], process.StoreFilePaths(&sourceFiles))

	if err != nil {
		log.Errorf("Error walking directory: %v", err)
		return
	}

	log.Infof("Started %v", "Paralel")

	process.CreateHashes(&sourceFiles, 8)
	elapsed1 := time.Since(timer)
	log.Infof("paralel took: %s for %v files", &elapsed1, len(sourceFiles))
	// #endregion paralel

	process.FindDuplicates(&sourceFiles)
	duplicates := process.GetDuplicates(&sourceFiles)
	flattenedDuplicates := process.GetFlattened(&duplicates)
	err1 := process.SaveAsCSV(flattenedDuplicates, myEnv["RESULT_FILE"])
	if err1 != nil {
		logger.PanicAndLog(err1)
	}
	// visuals.PrintDuplicates(sourceFiles)
}
