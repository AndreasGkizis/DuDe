package models

type DuDeFile struct {
	Filename        string
	Hash            string
	FullPath        string
	DuplicatesFound []DuDeFile
}

type ResultEntry struct {
	Filename          string
	FullPath          string
	DuplicateFilename string
	DuplicateFullPath string
}

type FileHash struct {
	FilePath string
	Hash     string
	ModTime  int64
	FileSize int64
}

// #region helper methods

func FindByPath(hashes []FileHash, filePath string) *FileHash {
	for i := range hashes {
		if hashes[i].FilePath == filePath {
			return &hashes[i]
		}
	}
	return nil
}

// #endregion
