package validation

// MockValidator is a test-only struct to simulate the Validator behavior
type MockValidator struct {
	ReadableDirFunc func(path string) error
	WritableDirFunc func(path string) error
}

func (m MockValidator) Resolve(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func (m MockValidator) ReadableDir(path string) error {
	if m.ReadableDirFunc != nil {
		return m.ReadableDirFunc(path)
	}
	return nil
}
func (m MockValidator) WritableDir(path string) error {
	if m.WritableDirFunc != nil {
		return m.WritableDirFunc(path)
	}
	return nil
}
