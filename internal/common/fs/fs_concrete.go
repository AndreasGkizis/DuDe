package fs

import (
	"os"
	"path/filepath"
)

type OS struct{}

func (OS) Exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (OS) IsDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (OS) CanRead(p string) bool {
	_, err := os.ReadDir(p)
	return err == nil
}

func (OS) CanWrite(p string) bool {
	testFile, err := os.CreateTemp(p, ".dude-write-test-*")
	if err != nil {
		return false
	}

	closeErr := testFile.Close()
	removeErr := os.Remove(testFile.Name())
	return closeErr == nil && removeErr == nil
}

func (OS) Parent(p string) string {
	return filepath.Dir(p)
}
