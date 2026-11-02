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

// DirectoryCount returns the number of directories configured for scanning.
func (e *ExecutionParams) DirectoryCount() int {
	return len(e.Directories)
}
