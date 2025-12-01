package fs

// MockFS is a test-only struct that implements the fs.FS interface.
type MockFS struct {
	ExistsFunc   func(path string) bool
	IsDirFunc    func(path string) bool
	CanReadFunc  func(path string) bool
	CanWriteFunc func(path string) bool
	ParentFunc   func(path string) string
}

func (m MockFS) Exists(path string) bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(path)
	}
	return false
}

func (m MockFS) IsDir(path string) bool {
	if m.IsDirFunc != nil {
		return m.IsDirFunc(path)
	}
	return false
}

func (m MockFS) CanRead(path string) bool {
	if m.CanReadFunc != nil {
		return m.CanReadFunc(path)
	}
	return false
}

func (m MockFS) CanWrite(path string) bool {
	if m.CanWriteFunc != nil {
		return m.CanWriteFunc(path)
	}
	return false
}

func (m MockFS) Parent(path string) string {
	if m.ParentFunc != nil {
		return m.ParentFunc(path)
	}
	return ""
}
