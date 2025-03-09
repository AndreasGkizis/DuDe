package common

const (
	Def = "default"

	ArgFilename = "arguments.txt"
	ResFilename = "results.csv"
	MemFilename = "memory.db"

	// cli flags
	DbgFlagName_long      = "debug"
	DbgFlagName           = "dbg"
	DbgFlagActiveValue    = "enable"
	DbgFlagNotActiveValue = "disable"

	ModeFlag         = "m"
	ModeFlag_long    = "mode"
	ModeSingleFolder = "sf"
	ModeDualFolder   = "df"

	SourceFlag      = "s"
	SourceFlag_long = "source"

	TargetFlag      = "t"
	TargetFlag_long = "target"

	MemDirFlag      = "c"
	MemDirFlag_long = "cache-dir"

	ResultDirFlag      = "r"
	ResultDirFlag_long = "results"

	ArgFilename_Mode      = "EXECUTION_MODE"
	ArgFilename_Dbg       = "DEBUG_MODE"
	ArgFilename_cacheDir  = "CACHE_FILE"
	ArgFilename_resDir    = "RESULT_FILE"
	ArgFilename_sourceDir = "SOURCE_DIR"
	ArgFilename_targetDir = "TARGET_DIR"
	ArgFileContent        = ArgFilename_sourceDir + `=<... your desired source full path...>` + "\n" +
		ArgFilename_targetDir + `=<... your desired target full path...>` + "\n" +
		ArgFilename_resDir + `=<... your desired result file full path...>` + "\n" +
		ArgFilename_cacheDir + `=<... your desired memory file full path...>` + "\n" +
		ArgFilename_Mode + `=<... your desired execution mode here use "` + ModeSingleFolder + `" for single-folder or "` + ModeDualFolder + `" for dual-folder ...>` + "\n" +
		ArgFilename_Dbg + `=<... to enable Debug mode add "` + DbgFlagActiveValue + `" here ...>`
)

var (
	ResultsHeader = []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
)
