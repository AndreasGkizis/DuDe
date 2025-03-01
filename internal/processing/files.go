package processing

import (
	common "DuDe/common"
	"DuDe/internal/visuals"
	"DuDe/models"
	"encoding/csv"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func FindFullFilePath(dir string, filename string) (string, error) {
	var foundPath string

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.Name() == filename {
			foundPath = path
			return filepath.SkipDir // Stop walking after finding the first occurrence
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	return foundPath, nil
}

func StoreFilePaths(result *[]models.FileHash) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			if os.IsNotExist(err) {
				visuals.DirDoesNotExistMessage(path)
			}
			return err
		}

		if !d.IsDir() {
			newFile := models.FileHash{
				FilePath: path,
			}
			*result = append(*result, newFile)
		}

		return nil
	}
}

func GetFileName(input string) string {
	if input == "" {
		return ""
	}

	parts := strings.Split(input, "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func CreateArgsFile() error {
	filename := common.ArgFilename
	basedir := "."

	targetsPath, _ := FindFullFilePath(basedir, filename)
	_, err := os.Stat(targetsPath)

	if os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			common.PanicAndLog(err)
		}
		defer file.Close()

		// Write default argument file
		_, err = file.WriteString(common.ArgFileContent)

		if err != nil {
			common.PanicAndLog(err)
			return err
		}

		visuals.ArgsFileNotFound()

	}
	return nil
}

func SaveResultsAsCSV(data []models.ResultEntry, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := common.GetResultsHeader()
	err = writer.Write(header)
	if err != nil {
		return err
	}

	// Write data
	for _, entry := range data {
		err = writer.Write([]string{
			entry.Filename,
			entry.FullPath,
			entry.DuplicateFilename,
			entry.DuplicateFullPath,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
