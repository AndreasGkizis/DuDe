package processing

import (
	logger "DuDe/common"
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
	filename := "arguments.txt"
	basedir := "."

	targetsPath, _ := FindFullFilePath(basedir, filename)
	_, err := os.Stat(targetsPath)

	if os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			logger.PanicAndLog(err)
		}
		defer file.Close()

		// Write default arguments to the file
		_, err = file.WriteString(
			`SOURCE_DIR=<... your desired source full path...>
TARGET_DIR=<... your desired target full path...>
RESULT_FILE=<... your desired result file full path...>
MEMORY_FILE=<... your desired memory file full path...>`)

		if err != nil {
			logger.PanicAndLog(err)
			return err
		}

		fmt.Printf("\nThe '%s' file was not found! So a NEW one has been created for you =].\n", filename)
		fmt.Print("Follow these steps:\n")
		fmt.Print("1. Open the newly created 'arguments.txt' file.\n")
		fmt.Print("2. Add the paths you want to the folders you want to scan.\n")
		fmt.Print("3. Save the file.\n")
		fmt.Print("4. Run the program again.\n")

		WaitAndExit()
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
	header := []string{"File Name", "Path", "Duplicate File Name", "Duplicate Path"}
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
		logger.GetLogger().Info("File already exists. Skipping creation.")
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
	header := []string{"File Path", "Hash", "Modification Time", "File Size"}
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
