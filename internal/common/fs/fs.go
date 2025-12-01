package fs

type FS interface {
	Exists(path string) bool
	IsDir(path string) bool
	CanRead(path string) bool
	CanWrite(path string) bool
	Parent(path string) string
}
