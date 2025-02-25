// db/database.go
package db

import (
	"DuDe/models/db_models"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DatabaseConfig struct {
	DSN string
}

// NewDatabase returns a new database connection
func NewDatabase(path string) (*gorm.DB, error) {

	// dbDir := filepath.Dir(path) // Extract the directory part of the path
	// if _, err := os.Stat(dbDir); os.IsNotExist(err) {
	// 	if err := os.MkdirAll(dbDir, 0755); err != nil { // Create directories recursively
	// 		return nil, err // Handle directory creation error
	// 	}
	// }

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	AutoMigrate(db)
	return db, nil
}

// AutoMigrate creates the database schema based on models
func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&db_models.FileHash{},
		// Add other models here as needed
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDatabase(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
