package unit_tests

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"DuDe/internal/common"
	database "DuDe/internal/db"
	"DuDe/internal/models/db_models"
)

func TestInitializeDatabaseCreatesExpectedSchema(t *testing.T) {
	directory := t.TempDir()
	db, err := database.InitializeDatabase(directory)
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	t.Cleanup(func() { closeDatabase(t, db) })

	if _, err := os.Stat(filepath.Join(directory, common.MemFilename)); err != nil {
		t.Fatalf("stat database file: %v", err)
	}

	rows, err := db.Query("PRAGMA table_info(file_hashes)")
	if err != nil {
		t.Fatalf("query schema: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var columnID int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&columnID, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan schema column: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate schema: %v", err)
	}

	for _, expected := range []string{"id", "path", "hash", "size", "modified_time", "updated_at", "created_at"} {
		if !columns[expected] {
			t.Errorf("expected column %q", expected)
		}
	}

	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("run migration twice: %v", err)
	}
}

func TestGetDatabaseConnectionReopensExistingDatabase(t *testing.T) {
	directory := t.TempDir()
	db, err := database.InitializeDatabase(directory)
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}

	repository := database.NewFileHashRepository(db)
	want := testFileHash("/files/one.txt", "hash-one")
	if err := repository.Create(want); err != nil {
		t.Fatalf("create record: %v", err)
	}
	closeDatabase(t, db)

	reopened, err := database.GetDatabaseConnection(directory)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() { closeDatabase(t, reopened) })

	got, err := database.NewFileHashRepository(reopened).GetByPath(want.FilePath)
	if err != nil {
		t.Fatalf("get reopened record: %v", err)
	}
	assertFileHashValues(t, got, want)
}

func TestFileHashRepositoryCreateAndRead(t *testing.T) {
	_, repository := newTestRepository(t)
	first := testFileHash("/files/one.txt", "hash-one")
	second := testFileHash("/files/two.txt", "hash-two")

	if err := repository.Create(first); err != nil {
		t.Fatalf("create first record: %v", err)
	}
	if err := repository.Create(second); err != nil {
		t.Fatalf("create second record: %v", err)
	}

	got, err := repository.GetByPath(first.FilePath)
	if err != nil {
		t.Fatalf("get record by path: %v", err)
	}
	assertFileHashValues(t, got, first)
	if !got.CreatedAt.Valid {
		t.Error("expected created_at to be populated")
	}

	all, err := repository.GetAll()
	if err != nil {
		t.Fatalf("get all records: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 records, got %d", len(all))
	}
}

func TestFileHashRepositoryCreateRejectsDuplicatePath(t *testing.T) {
	_, repository := newTestRepository(t)
	record := testFileHash("/files/duplicate.txt", "hash-one")

	if err := repository.Create(record); err != nil {
		t.Fatalf("create record: %v", err)
	}
	if err := repository.Create(record); err == nil {
		t.Fatal("expected duplicate path to return an error")
	}
}

func TestFileHashRepositoryGetByPathReturnsErrorWhenMissing(t *testing.T) {
	_, repository := newTestRepository(t)

	record, err := repository.GetByPath("/files/missing.txt")
	if !errors.Is(err, database.ErrFileHashNotFound) {
		t.Fatalf("expected ErrFileHashNotFound, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected no record, got %#v", record)
	}
}

func TestFileHashRepositoryUpdate(t *testing.T) {
	_, repository := newTestRepository(t)
	record := testFileHash("/files/update.txt", "old-hash")
	if err := repository.Create(record); err != nil {
		t.Fatalf("create record: %v", err)
	}

	record.Hash = "new-hash"
	record.FileSize = 2048
	record.ModTime = "2026-08-09T11:00:00Z"
	if err := repository.Update(record); err != nil {
		t.Fatalf("update record: %v", err)
	}

	got, err := repository.GetByPath(record.FilePath)
	if err != nil {
		t.Fatalf("get updated record: %v", err)
	}
	assertFileHashValues(t, got, record)
	if !got.UpdatedAt.Valid {
		t.Error("expected updated_at to be populated")
	}

	if err := repository.Update(record); err != nil {
		t.Fatalf("repeat unchanged update: %v", err)
	}
}

func TestFileHashRepositoryUpdateReturnsErrorWhenMissing(t *testing.T) {
	_, repository := newTestRepository(t)

	if err := repository.Update(testFileHash("/files/missing.txt", "hash")); !errors.Is(err, database.ErrFileHashNotFound) {
		t.Fatalf("expected ErrFileHashNotFound, got %v", err)
	}
}

func TestFileHashRepositoryUpsertCreatesAndUpdates(t *testing.T) {
	_, repository := newTestRepository(t)
	record := testFileHash("/files/upsert.txt", "first-hash")

	if err := repository.Upsert(record); err != nil {
		t.Fatalf("upsert new record: %v", err)
	}

	record.Hash = "second-hash"
	record.FileSize = 4096
	if err := repository.Upsert(record); err != nil {
		t.Fatalf("upsert existing record: %v", err)
	}

	got, err := repository.GetByPath(record.FilePath)
	if err != nil {
		t.Fatalf("get upserted record: %v", err)
	}
	assertFileHashValues(t, got, record)
}

func TestUpsertReturnsUnexpectedUpdateErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { closeDatabase(t, db) })

	_, err = db.Exec(`CREATE TABLE file_hashes (
		path TEXT UNIQUE,
		hash TEXT,
		size INTEGER,
		modified_time TEXT,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatalf("create malformed schema: %v", err)
	}

	repository := database.NewFileHashRepository(db)
	err = repository.Upsert(testFileHash("/files/error.txt", "hash"))
	if err == nil {
		t.Fatal("expected upsert to propagate the lookup error")
	}
}

func TestTruncateDatabaseRemovesAllRecords(t *testing.T) {
	db, repository := newTestRepository(t)
	if err := repository.Create(testFileHash("/files/one.txt", "hash")); err != nil {
		t.Fatalf("create record: %v", err)
	}

	if err := database.TruncateDatabase(db); err != nil {
		t.Fatalf("truncate database: %v", err)
	}

	records, err := repository.GetAll()
	if err != nil {
		t.Fatalf("get records after truncate: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no records, got %d", len(records))
	}
}

func TestDatabaseOperationsReturnErrorsAfterClose(t *testing.T) {
	db, err := database.InitializeDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	closeDatabase(t, db)

	if err := database.AutoMigrate(db); err == nil {
		t.Error("expected migration on closed database to fail")
	}
	if err := database.TruncateDatabase(db); err == nil {
		t.Error("expected truncate on closed database to fail")
	}
}

func TestDeleteDatabaseRemovesDatabaseFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, common.MemFilename)
	db, err := database.InitializeDatabase(directory)
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	closeDatabase(t, db)

	if err := database.DeleteDatabase(path); err != nil {
		t.Fatalf("delete database: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected database file to be removed, got %v", err)
	}
}

func newTestRepository(t *testing.T) (*sql.DB, *database.FileHashRepository) {
	t.Helper()

	db, err := database.InitializeDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	t.Cleanup(func() { closeDatabase(t, db) })
	return db, database.NewFileHashRepository(db)
}

func testFileHash(path, hash string) *db_models.FileHash {
	return &db_models.FileHash{
		FilePath: path,
		Hash:     hash,
		FileSize: 1024,
		ModTime:  "2026-08-09T10:00:00Z",
	}
}

func assertFileHashValues(t *testing.T, got, want *db_models.FileHash) {
	t.Helper()

	if got.FilePath != want.FilePath || got.Hash != want.Hash || got.FileSize != want.FileSize || got.ModTime != want.ModTime {
		t.Errorf("file hash mismatch: got %#v, want %#v", got, want)
	}
}

func closeDatabase(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Errorf("close database: %v", err)
	}
}
