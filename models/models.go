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

type FileHashCollection struct {
	Hashes map[string]FileHash
}

func (fhc *FileHashCollection) ToSlice() []FileHash {
	fileHashes := make([]FileHash, 0, len(fhc.Hashes))
	for _, fileHash := range fhc.Hashes {
		fileHashes = append(fileHashes, fileHash)
	}
	return fileHashes
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
