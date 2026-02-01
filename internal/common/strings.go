package common

const (
	Def = "default"

	Results_file_name      = "results"
	Results_file_extension = "csv"
	MemFilename            = "memory.db"

	ResultsFileSeperator = "------"

	ArgFilename_cacheDir     = "CACHE_FILE"
	ArgFilename_resDir       = "RESULT_FILE"
	ArgFilename_sourceDir    = "SOURCE_DIR"
	ArgFilename_targetDir    = "TARGET_DIR"
	ArgFilename_paranoidMode = "PARANOID"

	CLI_Intro = `
	██████╗  ██╗   ██╗        ██████╗  ███████╗ 
	██╔══██╗ ██║   ██║        ██╔══██╗ ██╔════╝ 
	██║  ██║ ██║   ██║ █████╗ ██║  ██║ █████╗   
	██║  ██║ ██║   ██║ ╚════╝ ██║  ██║ ██╔══╝   
	██████╔╝ ╚██████╔╝        ██████╔╝ ███████╗ 
	╚═════╝   ╚═════╝         ╚═════╝  ╚══════╝ 
	--------------------------------------------
	Welcome to Duplicate Detection         
	--------------------------------------------
	
	🔍 Let's find those duplicates...  
	💀 ..and....KILL 'EM!
	
	`
)

var (
	ResultsHeader = []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
)
