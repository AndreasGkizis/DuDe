// file_hash_repository_sqlite.go
package db

import (
	"DuDe/models/db_models"
	"errors"
	"time"

	"gorm.io/gorm"
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
	db *gorm.DB
}

func (r *FileHashRepository) GetAll() ([]*db_models.FileHash, error) {
	var filehashes []*db_models.FileHash
	result := r.db.Find(&filehashes)
	if result.Error != nil {
		return nil, result.Error
	}

	return filehashes, nil
}

func (r *FileHashRepository) GetByPath(path string) (*db_models.FileHash, error) {
	var filehash *db_models.FileHash
	result := r.db.Where(&db_models.FileHash{FilePath: path}).First(&filehash)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("no record found")
	}

	return filehash, nil
}

func (r *FileHashRepository) Create(fh *db_models.FileHash) error {
	result := r.db.Save(fh)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("did not save")
	}

	return nil
}

func (r *FileHashRepository) Update(fh *db_models.FileHash) error {
	existingFH, err := r.GetByPath(fh.FilePath)
	if err != nil {
		return err
	}

	existingFH.FilePath = fh.FilePath
	existingFH.Hash = fh.Hash
	existingFH.FileSize = fh.FileSize
	existingFH.ModTime = fh.ModTime

	existingFH.UpdatedAt = time.Now()

	result := r.db.Save(existingFH)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
