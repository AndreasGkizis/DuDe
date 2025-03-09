package db_models

import "database/sql"

type FileHash struct {
	ID       uint
	FilePath string
	Hash     string
	FileSize int64
	ModTime  string

	// helpers
	CreatedAt sql.NullString
	UpdatedAt sql.NullString
}
