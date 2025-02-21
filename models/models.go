package models

import (
	"time"
)

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

type FileHashBatch struct {
	ID        int
	Entries   []FileHash
	BatchSize uint
	Saved     bool
}

func NewFileBatch(batchSize uint) *FileHashBatch {
	return &FileHashBatch{
		ID:        int(time.Now().UnixNano()),     // supposed to be pretty unique for my needs
		Entries:   make([]FileHash, 0, batchSize), // Length 0, capacity maxEntries
		BatchSize: batchSize,
		Saved:     false,
	}
}

type FileHashCollection struct {
	Hashes map[string]FileHash
}

type FileHashSlice []FileHash

func (collection *FileHashCollection) ToSlice() []FileHash {
	fileHashes := make([]FileHash, 0, len(collection.Hashes))
	for _, fileHash := range collection.Hashes {
		fileHashes = append(fileHashes, fileHash)
	}
	return fileHashes
}

// #region helper methods

func (collection FileHashSlice) FindByHash(hash string) *FileHash {
	for _, fileHash := range collection {
		if fileHash.Hash == hash {
			return &fileHash
		}
	}
	return nil
}

func FindByPath(hashes []FileHash, filePath string) *FileHash {
	for i := range hashes {
		if hashes[i].FilePath == filePath {
			return &hashes[i]
		}
	}
	return nil
}

// #endregion
