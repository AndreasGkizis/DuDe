package models

// SkippedPath records a filesystem entry that DuDe could not inspect during
// discovery. It is diagnostic information and is not a missing source file.
type SkippedPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ScanDiagnostics describes non-fatal filesystem entries skipped during a
// completed scan.
type ScanDiagnostics struct {
	SkippedSourcePaths []SkippedPath `json:"skippedSourcePaths"`
	SkippedTargetPaths []SkippedPath `json:"skippedTargetPaths"`
}
