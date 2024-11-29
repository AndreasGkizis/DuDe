package processing

import (
	logger "DuDe/common"
	"DuDe/models"
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

func CalculateMD5Hash(file models.DuDeFile) (string, error) {
	hasherMD5 := md5.New()

	f, err := os.Open(file.FullPath)
	defer func() {
		err := f.Close()
		if err != nil {
			logger.PanicAndLog(err)
		}
	}()

	io.Copy(hasherMD5, f)
	return fmt.Sprintf("%x", hasherMD5.Sum(nil)), err
}

func FindDuplicates(input *[]models.DuDeFile) {
	for i := range *input {
		occuranceCounter := 0
		for j := range *input {
			if (*input)[i].Hash == (*input)[j].Hash {
				if occuranceCounter == 0 {
					occuranceCounter++
				} else {
					(*input)[i].DuplicatesFound = append((*input)[i].DuplicatesFound, (*input)[j])
				}
			}
		}
	}
}

func GetDuplicates(input *[]models.DuDeFile) []models.DuDeFile {
	seen := make(map[string]bool)
	result := make([]models.DuDeFile, 0)
	for _, val := range *input {
		if len(val.DuplicatesFound) > 0 {
			if !seen[val.Hash] {
				seen[val.Hash] = true
				result = append(result, val)
			}
		}
	}
	return result
}

func GetFlattened(input *[]models.DuDeFile) []models.ResultEntry {
	result := make([]models.ResultEntry, 0)
	for _, val := range *input {
		for _, dup := range val.DuplicatesFound {
			a := models.ResultEntry{
				Filename:          val.Filename,
				FullPath:          val.FullPath,
				DuplicateFilename: dup.Filename,
				DuplicateFullPath: dup.FullPath,
			}
			result = append(result, a)
		}
	}
	return result
}
