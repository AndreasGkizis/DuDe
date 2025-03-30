package processing

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
	"encoding/csv"
	"fmt"
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
			log.Logger.DPanic(err)
		}
		defer file.Close()
		content := []string{
			common.FileIntro,
			common.Exmaple_FileArg_Usage,
			common.FileOutro,
			common.ArgFileSettigns,
		}

		for _, text := range content {
			if _, err := file.WriteString(text); err != nil {
				log.Logger.Fatal(err)
			}
		}
	}

	return fullfilepath
}

func SaveResultsAsCSV(data []models.ResultEntry, fullpath string) error {
	log.InfoWithFuncName(fmt.Sprintf("%d results found ", len(data)))
	log.InfoWithFuncName(fmt.Sprintf("creating results file in path :%s", fullpath))
	file, err := os.Create(fullpath)
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
