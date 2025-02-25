// db/database.go
package db

import (
	"DuDe/models/db_models"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig struct {
	DSN string
}

// NewDatabase returns a new database connection
func NewDatabase(path string, debugEnabled bool) (*gorm.DB, error) {

	var db *gorm.DB
	var err error

	if debugEnabled {
		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      false,       // Don't include params in the SQL log
				Colorful:                  true,        // Disable color

			},
		)

		db, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: newLogger})

	} else {
		db, err = gorm.Open(sqlite.Open(path), &gorm.Config{})

	}
	if err != nil {
		return nil, err
	}

	if err = AutoMigrate(db); err != nil { //Move AutoMigrate into the function and handle the error.
		return nil, err
	}

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
