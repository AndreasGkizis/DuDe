package models

type ResultEntry struct {
	Filename          string
	FullPath          string
	DuplicateFilename string
	DuplicateFullPath string
}

type FileHash struct {
	FileName        string
	FilePath        string
	Hash            string
	ModTime         string
	FileSize        int64
	DuplicatesFound []FileHash
}

// TODO This should remain immutable!!not sure how to force this yet
type ExecutionParams struct {
	Directories  []string `json:"directories"`
	UseCache     bool     `json:"useCache"`
	CacheDir     string   `json:"cacheDir"`
	ResultsDir   string   `json:"resultsDir"`
	ParanoidMode bool     `json:"paranoidMode"`
	CPUs         int      `json:"cpus"`
	BufSize      int      `json:"bufSize"`
	DebugMode    bool     `json:"debugMode"`
}

// DirectoryCount returns the number of directories configured for scanning.
func (e *ExecutionParams) DirectoryCount() int {
	return len(e.Directories)
}
