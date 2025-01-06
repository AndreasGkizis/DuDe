package processing

import (
	common "DuDe/common"
	"DuDe/internal/visuals"
	"DuDe/models"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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

func StoreFilePaths(result *[]models.DuDeFile) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			if os.IsNotExist(err) {
				visuals.DirDoesNotExistMessage(path)
			}
			return err
		}

		if !d.IsDir() {
			newFile := models.DuDeFile{
				FullPath: path,
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

func CreateMemoryCSV(filename string) error {

	_, err := os.Stat(filename)
	if err == nil {
		// File exists, no need to create it again
		common.GetLogger().Info("File already exists. Skipping creation.")
		return nil
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := common.GetMemHeader()
	err = writer.Write(header)
	if err != nil {
		return err
	}
	return nil
}

func AddSingleToMemoryCSV(memoryPath string, info models.FileHash) error {

	f, err := os.OpenFile(memoryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// headers => "File Path", "Hash", "Modification Time", "File Size"
	err = writer.Write([]string{
		info.FilePath,
		info.Hash,
		fmt.Sprintf("%d", info.ModTime),
		fmt.Sprintf("%d", info.FileSize),
	})

	if err != nil {
		return err
	}
	return nil
}

func WriteManyToMemoryCSV(memoryPath string, info []models.FileHash) error {

	f, err := os.OpenFile(memoryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	for _, v := range info {
		AddSingleToMemoryCSV(memoryPath, v)
	}

	return nil

}

func UpsertMemoryCSV(memoryPath string, info []models.FileHash) error {

	f, err := os.OpenFile(memoryPath, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	WriteManyToMemoryCSV(memoryPath, info)
	return nil
}

func LoadMemoryCSV(filepath string) ([]models.FileHash, error) {
	result := make([]models.FileHash, 0)
	f, err := os.Open(filepath)

	if err != nil {
		if os.IsNotExist(err) {
			visuals.DirDoesNotExistMessage(filepath)
		}
		return nil, err
	}

	defer f.Close()

	reader := csv.NewReader(f)

	_, err = reader.Read() // Read and discard the first row (header)
	if err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if err == nil {
			modTime, _ := strconv.ParseInt(record[2], 10, 64)
			fileSize, _ := strconv.ParseInt(record[3], 10, 64)
			fileHash := models.FileHash{
				FilePath: record[0],
				Hash:     record[1],
				ModTime:  modTime,
				FileSize: fileSize,
			}

			result = append(result, fileHash)
		} else if errors.Is(err, io.EOF) {
			return result, nil
		}
	}
}
