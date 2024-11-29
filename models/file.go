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
