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
	// hasherMD5 := md5.New()
	// var doneFiles float64 = 0

	return func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			f, _ := os.Open(path)
			defer f.Close()
			// io.Copy(hasherMD5, f)
			// md5hash := hasherMD5.Sum(nil)

			newFile := models.DuDeFile{
				FullPath: path,
				// Hash:     fmt.Sprintf("%x", md5hash),
			}
			*result = append(*result, newFile)
			// myHashMap[path] = fmt.Sprintf("%x", md5hash)

			// hasherMD5.Reset()
			// doneFiles++

			// if *enableBenchmark {
			// 	elapsed := time.Since(timer)
			// 	fileInfo, _ := os.Stat(path)
			// 	fileSizeMB := float64(fileInfo.Size()) / (1 << 20) // Convert to MB (1 << 20 = 1048576)
			// 	log.Debug().Msgf("\nFilename: %s | Size: %0.2f MB | Took: %v | MD5: %s\n", d.Name(), fileSizeMB, elapsed, file.Filename)
			// }
		}

		return nil
	}
}

func GetFileName(input string) string {
	parts := strings.Split(input, "/")
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
