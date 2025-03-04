package processing

import (
	common "DuDe/common"
	"DuDe/internal/visuals"
	"DuDe/models"
	"encoding/csv"
	"io/fs"
	"os"
	"path/filepath"
)

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

func CreateArgsFile() error {
	executablePath, err := os.Executable()
	baseDir := filepath.Dir(executablePath)
	fullfilepath := filepath.Join(baseDir, common.ArgFilename)

	if err != nil {
		common.PanicAndLog(err)
	}
	_, err = os.Stat(fullfilepath)

	if os.IsNotExist(err) {
		file, err := os.Create(fullfilepath)
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

	err = writer.Write(common.ResultsHeader)
	if err != nil {
		return err
	}

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
