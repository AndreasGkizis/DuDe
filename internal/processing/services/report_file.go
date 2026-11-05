package services

import (
	"DuDe/internal/common"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// createReportFile retries timestamped names with numeric suffixes on collision.
// create must use exclusive creation so existing reports are never overwritten.
func createReportFile(directory, prefix string, create func(string) (io.WriteCloser, error)) (string, io.WriteCloser, error) {
	timestamp := time.Now().Format("_2006_01_02_15_04_05")
	for suffix := 0; ; suffix++ {
		name := fmt.Sprintf("%s%s.%s", prefix, timestamp, common.Results_file_extension)
		if suffix > 0 {
			name = fmt.Sprintf("%s%s_%d.%s", prefix, timestamp, suffix, common.Results_file_extension)
		}
		path := filepath.Join(directory, name)
		file, err := create(path)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		return path, file, nil
	}
}
