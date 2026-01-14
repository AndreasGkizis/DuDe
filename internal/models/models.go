package models

import "DuDe/internal/common"

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
	SourceDir             string `json:"sourceDir"`
	TargetDir             string `json:"targetDir"`
	UseCache              bool   `json:"useCache"`
	CacheDir              string `json:"cacheDir"`
	ResultsDir            string `json:"resultsDir"`
	ParanoidMode          bool   `json:"paranoidMode"`
	CPUs                  int    `json:"cpus"`
	BufSize               int    `json:"bufSize"`
	DebugMode             bool   `json:"debugMode"`
	DualFolderModeEnabled bool   `json:"dualFolderModeEnabled"`
}

func (e *ExecutionParams) IsDualFolderMode() bool {
	return ((e.TargetDir != "" && e.TargetDir != common.Def) && (e.SourceDir != "" && e.SourceDir != common.Def))
}
