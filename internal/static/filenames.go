package static

import "sync"

var (
	mu          sync.Mutex
	argFilename string
	memFilename string
	resFilename string

	argFilename_memDir    string
	argFilename_resDir    string
	argFilename_sourceDir string
	argFilename_targetDir string
)

func init() {
	argFilename = "arguments.txt"
	memFilename = "memory.csv"
	resFilename = "results.csv"

	argFilename_memDir = "MEMORY_FILE"
	argFilename_resDir = "RESULT_FILE"
	argFilename_sourceDir = "SOURCE_DIR"
	argFilename_targetDir = "TARGET_DIR"
}

func GetArgFilename() string {
	mu.Lock()
	defer mu.Unlock()
	return argFilename
}

func GetMemFilename() string {
	mu.Lock()
	defer mu.Unlock()
	return memFilename
}

func GetResFilename() string {
	mu.Lock()
	defer mu.Unlock()
	return resFilename
}

func GetMemDirTag() string {
	mu.Lock()
	defer mu.Unlock()
	return argFilename_memDir
}

func GetSourceDirTag() string {
	mu.Lock()
	defer mu.Unlock()
	return argFilename_sourceDir
}

func GetResultDirTag() string {
	mu.Lock()
	defer mu.Unlock()
	return argFilename_resDir
}

func GetTargetDirTag() string {
	mu.Lock()
	defer mu.Unlock()
	return argFilename_targetDir
}
