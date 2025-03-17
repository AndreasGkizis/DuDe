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

type ExecutionParams struct {
	SourceDir  string
	TargetDir  string
	CacheDir   string
	ResultsDir string
	Cpus       int
	BufSize    int
}

// #region helper methods

func FindByPath(fhs *[]FileHash, filePath string) *FileHash {
	for i := range *fhs {
		if (*fhs)[i].FilePath == filePath {
			return &(*fhs)[i]
		}
	}
	return nil
}

// #endregion
