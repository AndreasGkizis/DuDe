package db_models

import "gorm.io/gorm"

type FileHash struct {
	gorm.Model
	FilePath string `gorm:"uniqueIndex;not null"`
	Hash     string `gorm:"not null"`
	ModTime  int64  `gorm:"not null"`
	FileSize int64  `gorm:"not null"`
}
