package common

import "sync"

const (
	Def = "default"

	ArgFilename = "arguments.txt"
	ResFilename = "results.csv"

	// cli flags
	DbgFlagName_long   = "debug"
	DbgFlagName        = "dbg"
	DbgFlagActiveValue = "enable"

	ModeFlag      = "m"
	ModeFlag_long = "mode"

	SourceFlag      = "s"
	SourceFlag_long = "source"

	TargetFlag      = "t"
	TargetFlag_long = "target"

	CacheDirFlag      = "c"
	CacheDirFlag_long = "cache-dir"

	ResultDirFlag      = "r"
	ResultDirFlag_long = "results"

	ArgFilename_cacheDir  = "CACHE_FILE"
	ArgFilename_resDir    = "RESULT_FILE"
	ArgFilename_sourceDir = "SOURCE_DIR"
	ArgFilename_targetDir = "TARGET_DIR"
	ArgFileContent        = `SOURCE_DIR=<... your desired source full path...>
TARGET_DIR=<... your desired target full path...>
RESULT_FILE=<... your desired result file full path...>
CACHE_FILE=<... your desired memory file full path...>`
)

var (
	mu            sync.Mutex
	resultsHeader = []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
)

func GetResultsHeader() []string {
	mu.Lock()
	defer mu.Unlock()
	return resultsHeader
}
