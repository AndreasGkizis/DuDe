package common

import "sync"

const (
	ArgFilename = "arguments.txt"
	MemFilename = "memory.csv"
	ResFilename = "results.csv"

	ArgFilename_memDir    = "MEMORY_FILE"
	ArgFilename_resDir    = "RESULT_FILE"
	ArgFilename_sourceDir = "SOURCE_DIR"
	ArgFilename_targetDir = "TARGET_DIR"
)

var (
	mu            sync.Mutex
	memoryHeader  = []string{"File Path", "Hash", "Modification Time", "File Size"}
	resultsHeader = []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
)

func GetMemHeader() []string {
	mu.Lock()
	defer mu.Unlock()
	return memoryHeader
}

func GetResultsHeader() []string {
	mu.Lock()
	defer mu.Unlock()
	return resultsHeader
}
