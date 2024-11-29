package handlers

import (
	"encoding/csv"
	"os"
	"path/filepath"
)

func MakeOutputFile(filePath string) (*os.File, error) {
	csvHeaders := []string{"Filename", "Duplicate"}
	finalFilePath := filepath.Join("", filePath)
	file, err := os.OpenFile(finalFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	// defer file.Close()

	csvwriter := csv.NewWriter(file)
	defer csvwriter.Flush()

	if err := csvwriter.Write(csvHeaders); err != nil {
		panic(err)
	}

	return file, nil
}
