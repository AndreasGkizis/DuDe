package db

import (
	"DuDe/internal/common"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitializeDatabase returns a new database connection
func InitializeDatabase(dir string) (*sql.DB, error) {
	dbpath := filepath.Join(dir, common.MemFilename)

	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		return nil, err
	}

	if err = AutoMigrate(db); err != nil {
		db.Close() // Close the db if automigration fails.
		return nil, err
	}

	return db, nil
}
func GetDatabaseConnection(dir string) (*sql.DB, error) {
	dbpath := filepath.Join(dir, common.MemFilename)

	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
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

	if err != nil {
		return err
	}
	return nil
}

func TruncateDatabase(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM file_hashes")
	return err
}

// Removes db file, currently unsed but maybe useful
func DeleteDatabase(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
