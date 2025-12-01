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
	test := filepath.Join(p, ".tmp_write_test")
	f, err := os.Create(test)
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(test)
	return true
}

func (OS) Parent(p string) string {
	return filepath.Dir(p)
}
