package processing

import (
	common "DuDe-wails/internal/common"
	log "DuDe-wails/internal/common/logger"
	models "DuDe-wails/internal/models"
	visuals "DuDe-wails/internal/visuals"

	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func WalkDir(path string, result *sync.Map, pt *visuals.ProgressCounter) {
	defer func() {
		pt.SenderFinished()
	}()

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started walking directory %s files", groupID, path))

	err := filepath.WalkDir(path, storeFilePaths(result, pt))

	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished walking directory %s files", groupID, path))
}

func CreateArgsFile() string {

	entrypoint := common.GetExecutableDir()
	fullfilepath := filepath.Join(entrypoint, common.ArgFilename)
	_, err := os.Stat(fullfilepath)

	if os.IsNotExist(err) {
		file, err := os.Create(fullfilepath)
		if err != nil {
			log.ErrorWithFuncName(err.Error())
		}
		defer file.Close()
		content := []string{
			common.FileIntro,
			common.Example_FileArg_Usage,
			common.FileOutro,
			common.ArgFileSettings,
		}

		for _, text := range content {
			if _, err := file.WriteString(text); err != nil {
				log.ErrorWithFuncName(err.Error())
			}
		}

	}

	return fullfilepath
}

func SaveResultsAsCSV(data []models.ResultEntry, fullpath string) error {
	log.InfoWithFuncName(fmt.Sprintf("%d results found ", len(data)))
	log.InfoWithFuncName(fmt.Sprintf("creating results file in path :%s", fullpath))

	if len(data) == 0 {
		return nil
	}

	file, err := os.Create(fullpath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Comma = GetDelimiterForOS()

	// Write the UTF-8 BOM bytes at the very beginning of the file to force stupid excel to recognise the encoding.
	_, err = file.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		return fmt.Errorf("failed to write UTF-8 BOM: %v", err)
	}

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

func storeFilePaths(result *sync.Map, pt *visuals.ProgressCounter) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				visuals.DirDoesNotExistMessage(path)
			} else if errors.Is(err, os.ErrPermission) {
				log.WarnWithFuncName(fmt.Sprintf("skipping from err check: %s reason: %s", path, err.Error())) // wont work?
				return filepath.SkipDir                                                                        // Skip without failing
			} else {
				log.ErrorWithFuncName(fmt.Sprintf("skipping from err check: %s reason: %s", path, err.Error())) // wont work?
				return filepath.SkipDir                                                                         // Skip without failing
			}
			// return err
		}

		if !d.IsDir() {

			result.Store(path, models.FileHash{FilePath: path})
			pt.Channel <- 1
		}
		return nil
	}
}

func GetDelimiterForOS() rune {
	var delimiter rune
	if runtime.GOOS == "windows" {
		delimiter = ';'
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	} else {
		delimiter = ',' // Default for Linux, macOS, etc.
		log.InfoWithFuncName(fmt.Sprintf("Using (%c) delimiter for %s default.", delimiter, runtime.GOOS))
	}
	return delimiter
}
