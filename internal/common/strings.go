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
	Path_prefix        = "["
	Path_suffix        = "]"

	ArgFilename_Mode      = "EXECUTION_MODE"
	ArgFilename_Dbg       = "DEBUG_MODE"
	ArgFilename_cacheDir  = "CACHE_FILE"
	ArgFilename_resDir    = "RESULT_FILE"
	ArgFilename_sourceDir = "SOURCE_DIR"
	ArgFilename_targetDir = "TARGET_DIR"
	ArgFileSettigns       = `
` + ArgFilename_sourceDir + `=` + Path_prefix + `replace this text with your source path` + Path_suffix + "\n" +
		ArgFilename_targetDir + `=` + Path_prefix + `replace this text with your target path (optional)` + Path_suffix + "\n" +
		ArgFilename_resDir + `=` + Path_prefix + `replace this text with the path where the results file will be created (optional)` + Path_suffix

	FileIntro = `── ❖ ── How to Use this Configuration File ── ❖ ──

This program helps you find duplicate files in one or two folders. To use it, you need to provide some details in this text file before running the program.

This Program DOES NOT EDIT YOUR FILES! you can run it as many times as needed.

The resulting file

── ✷ ── What to Enter in the File ── ✷ ──

`
	Exmaple_FileArg_Usage = `1. ` + ArgFilename_sourceDir + ` – The main folder where you want to check for duplicate files.
   - Example: ` + ArgFilename_sourceDir + `=C:\Users\John\Documents

2. ` + ArgFilename_targetDir + ` (Optional) – A second folder to compare with the first one. If left empty, the program will only check for duplicates within the SOURCE_DIR.
   - Example: ` + ArgFilename_targetDir + `=D:\Backup\Documents
   - If you don’t need a second folder, ingore this setting.

3. ` + ArgFilename_resDir + `(Optional) – The folder where the program will save the list of duplicate files.
   - Example: ` + ArgFilename_resDir + `=C:\Users\John\Desktop
   - If you don’t set a path the file will be created in the same folder as the executable (DuDe.exe)

`

	FileOutro = `── ✶ ── Running the Program ── ✶ ──  

After setting up the file, save it and run the program. It will scan the folders and create a list of duplicate files in the ` + ResFilename + `.
This is a common filetype which can be opened in programs like Excel or LibreOffice or even plain old notepad.

── ✺ ── Enter Your Settings Below ── ✺ ──  
`

	CLI_Intro = `
	██████╗  ██╗   ██╗        ██████╗  ███████╗ 
	██╔══██╗ ██║   ██║        ██╔══██╗ ██╔════╝ 
	██║  ██║ ██║   ██║ █████╗ ██║  ██║ █████╗   
	██║  ██║ ██║   ██║ ╚════╝ ██║  ██║ ██╔══╝   
	██████╔╝ ╚██████╔╝        ██████╔╝ ███████╗ 
	╚═════╝   ╚═════╝         ╚═════╝  ╚══════╝ 
	--------------------------------------------
	Welcome to Duplicate Detection CLI         
	--------------------------------------------
	
	🔍 Let's find those duplicates...  
	💀 ..and....KILL 'EM!
	
	`
)

var (
	ResultsHeader = []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
)
