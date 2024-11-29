package processing

import (
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

func StoreFilePaths(result *[]models.DuDeFile) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			f, _ := os.Open(path)
			defer f.Close()
			newFile := models.DuDeFile{
				FullPath: path,
			}
			*result = append(*result, newFile)
		}

		return nil
	}
}

// func PopulateFilenames(sourceFiles *[]models.DuDeFile) {
// 	for index := range *sourceFiles {
// 		(*sourceFiles)[index].Filename = getFileName((*sourceFiles)[index].FullPath)
// 	}
// }

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

func SaveAsCSV(data []models.ResultEntry, filename string) error {
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
