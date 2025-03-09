package db

import (
	"DuDe/internal/models/db_models"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type FileHashRepo interface {
	GetByPath(path string) (*db_models.FileHash, error)
	GetAll() ([]*db_models.FileHash, error)
	Create(fh *db_models.FileHash) error
	Update(fh *db_models.FileHash) error
	Delete(id int) error
	DeleteByPath(path string) error
}

type FileHashRepository struct {
	Db *sql.DB
}

func NewFileHashRepository(db *sql.DB) *FileHashRepository {
	return &FileHashRepository{Db: db}
}

func (r *FileHashRepository) GetAll() ([]*db_models.FileHash, error) {
	var filehashes []*db_models.FileHash
	rows, err := r.Db.Query(`SELECT id, path, hash, size, modified_time, created_at FROM file_hashes`)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		filehash := &db_models.FileHash{}
		if err := rows.Scan(&filehash.ID, &filehash.FilePath, &filehash.Hash, &filehash.FileSize, &filehash.ModTime, &filehash.CreatedAt); err != nil {
			return nil, err
		}
		filehashes = append(filehashes, filehash)
	}

	if err := rows.Err(); err != nil { //check for errors from rows.next()
		return nil, err
	}

	return filehashes, nil
}

func (r *FileHashRepository) GetByPath(path string) (*db_models.FileHash, error) {
	filehash := &db_models.FileHash{}
	row := r.Db.QueryRow("SELECT id, path, hash, size, modified_time, updated_at, created_at FROM file_hashes WHERE path = ?", path)
	err := row.Scan(&filehash.ID, &filehash.FilePath, &filehash.Hash, &filehash.FileSize, &filehash.ModTime, &filehash.UpdatedAt, &filehash.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no record found")
		}
		return nil, err
	}
	return filehash, nil
}

func (r *FileHashRepository) Create(fh *db_models.FileHash) error {
	result, err := r.Db.Exec("INSERT INTO file_hashes (path, hash, size, modified_time, created_at) VALUES (?, ?, ?, ?, ?)",
		fh.FilePath, fh.Hash, fh.FileSize, fh.ModTime, time.Now().UTC().Format(time.RFC3339))

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return fmt.Errorf("error getting affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("did not save")
	}

	return nil
}

func (r *FileHashRepository) Update(fh *db_models.FileHash) error {
	var result sql.Result
	existingFH, err := r.GetByPath(fh.FilePath)

	if existingFH == nil {
		return errors.New("not found")
	}

	if fh.FilePath == existingFH.FilePath &&
		fh.FileSize == existingFH.FileSize &&
		fh.Hash == existingFH.Hash &&
		fh.ModTime == existingFH.ModTime {
		//do nothing
		return nil
	}

	if err != nil {
		return err
	}

	result, err = r.Db.Exec(`UPDATE file_hashes SET 
		hash = ?, 	size = ?, 		modified_time = ? ,updated_at =?		WHERE 	id = ?`,
		fh.Hash, fh.FileSize, fh.ModTime, time.Now().UTC().Format(time.RFC3339), existingFH.ID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return fmt.Errorf("error getting affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("did not save")
	}

	return nil
}

func (r *FileHashRepository) Upsert(fh *db_models.FileHash) error {
	err := r.Update(fh)

	if err != nil && err.Error() == "not found" {
		err := r.Create(fh)
		if err != nil {
			return err
		}
	}
	if err != nil && err.Error() == "did not save" {
		return err
	}

	return nil

}
