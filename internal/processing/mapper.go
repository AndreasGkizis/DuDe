package processing

import (
	"DuDe/internal/models"
	"DuDe/internal/models/db_models"
	"path/filepath"
)

func MapToServiceDTO(db_fh *db_models.FileHash) models.FileHash {
	return models.FileHash{
		FileName: filepath.Base(db_fh.FilePath),
		FilePath: db_fh.FilePath,
		Hash:     db_fh.Hash,
		ModTime:  db_fh.ModTime,
		FileSize: db_fh.FileSize,
	}
}

func MapToDomainDTO(ser_fh models.FileHash) db_models.FileHash {
	return db_models.FileHash{
		FilePath: ser_fh.FilePath,
		Hash:     ser_fh.Hash,
		ModTime:  ser_fh.ModTime,
		FileSize: ser_fh.FileSize,
	}
}
