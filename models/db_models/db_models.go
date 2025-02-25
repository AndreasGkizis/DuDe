package db_models

import (
	"time"

	"gorm.io/gorm"
)

type FileHash struct {
	ID        uint           `gorm:"primarykey;column:id"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
	FilePath  string         `gorm:"column:file_path;uniqueIndex;not null"`
	Hash      string         `gorm:"column:hash_value;not null"`
	ModTime   int64          `gorm:"column:modification_time;not null"`
	FileSize  int64          `gorm:"column:file_size;not null"`
}

func (fh *FileHash) GetUpdatefields() map[string]interface{} {

	// NOTE [ag]: this stupid thingy is for the upsert maybe there is a better way of doing this
	columnsToUpdate := make(map[string]interface{})
	columnsToUpdate["updated_at"] = time.Now()
	columnsToUpdate["file_path"] = fh.FilePath
	columnsToUpdate["hash_value"] = fh.Hash
	columnsToUpdate["modification_time"] = fh.ModTime
	columnsToUpdate["file_size"] = fh.FileSize

	return columnsToUpdate
}
