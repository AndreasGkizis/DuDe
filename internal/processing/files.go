package processing

import (
	common "DuDe/common"
	visuals "DuDe/internal/visuals"
	models "DuDe/models"
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

func CreateArgsFile() string {

	entrypoint := common.GetExecutableDir()
	fullfilepath := filepath.Join(entrypoint, common.ArgFilename)
	_, err := os.Stat(fullfilepath)

	if os.IsNotExist(err) {
		file, err := os.Create(fullfilepath)
		if err != nil {
			common.Logger.DPanic(err)
		}
		defer file.Close()

		_, err = file.WriteString(common.ArgFileContent)

		if err != nil {
			common.Logger.DPanic(err)
		}
	}

	return fullfilepath
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
