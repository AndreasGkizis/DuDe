package db

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

// NewDatabase returns a new database connection
func NewDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err = AutoMigrate(db); err != nil {
		db.Close() // Close the db if automigration fails.
		return nil, err
	}

	return db, nil
}

// AutoMigrate creates the database schema based on models
func AutoMigrate(db *sql.DB) error {
	_, err := db.Exec(`
                CREATE TABLE IF NOT EXISTS file_hashes (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        path TEXT UNIQUE,
                        hash TEXT,
                        size INTEGER,
                        modified_time TEXT,
						updated_at TEXT,
						created_at TEXT
                )
        `)

	// NOTE: add more tables as you go
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
