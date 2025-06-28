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

type ExecutionParams struct {
	SourceDir             string
	TargetDir             string
	CacheDir              string
	ResultsDir            string
	ParanoidMode          bool
	CPUs                  int
	BufSize               int
	DualFolderModeEnabled bool
}

func (e *ExecutionParams) IsDualFolderMode() bool {
	return ((e.TargetDir != "" && e.TargetDir != common.Def) && (e.SourceDir != "" && e.SourceDir != common.Def))
}
