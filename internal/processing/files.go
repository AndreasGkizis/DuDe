package processing

import (
	common "DuDe/internal/common"
	log "DuDe/internal/common/logger"
	models "DuDe/internal/models"
	visuals "DuDe/internal/visuals"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
)

func WalkDir(path string, result *map[string]models.FileHash, pt *visuals.ProgressCounter) {
	defer func() {
		pt.SenderFinished()
	}()

	groupID := rand.Uint32()
	log.InfoWithFuncName(fmt.Sprintf("Group %d started walking directory %s files", groupID, path))

	err := filepath.WalkDir(path, StoreFilePaths(result, pt))

	if err != nil {
		log.ErrorWithFuncName(fmt.Sprintf("Error walking directory: %v", err))
	}
	log.InfoWithFuncName(fmt.Sprintf("Group %d finished walking directory %s files", groupID, path))
}

func StoreFilePaths(result *map[string]models.FileHash, pt *visuals.ProgressCounter) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				visuals.DirDoesNotExistMessage(path)
			} else if errors.Is(err, os.ErrPermission) {
				log.WarnWithFuncName(fmt.Sprintf("skipping from err check: %s reason: %s", path, err.Error())) // wont work?
				return filepath.SkipDir                                                                        // Skip without failing
			}
			return err
		}

		if !d.IsDir() {
			// Check if we have read access to the file
			// bla := isFileReadable(path)
			// if bla == false {
			// 	log.WarnWithFuncName(fmt.Sprintf("skipping due to no read access: %s reason: %s", path, err.Error()))
			// 	return nil // Skip this file, don't propagate the error to stop the walk
			// }

			(*result)[path] = models.FileHash{FilePath: path}
			// *result = append(*result, models.FileHash{FilePath: path})
			pt.Channel <- 1
		}
		return nil
	}
}

func isFileReadable(filepath string) bool {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}

	mode := fileInfo.Mode()

	// Check specific permissions
	isReadable := mode.Perm()&0400 != 0
	// isWritable := mode.Perm() & 0200 != 0
	// isExecutable := mode.Perm() & 0100 != 0
	return isReadable
	// fmt.Printf("Readable: %v\n", isReadable)
	// fmt.Printf("Writable: %v\n", isWritable)
	// fmt.Printf("Executable: %v\n", isExecutable)
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
			common.Exmaple_FileArg_Usage,
			common.FileOutro,
			common.ArgFileSettigns,
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

	file, err := os.Create(fullpath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Comma = getDelimiterForOS()

	// Write the UTF-8 BOM bytes at the very beginning of the file to force stupid excelk to recognise the encoding.
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

func getDelimiterForOS() rune {
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

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
