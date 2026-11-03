package models

// MissingSourceFile describes one source file that has no equivalent target file.
type MissingSourceFile struct {
	Filename string `json:"filename"`
	FullPath string `json:"fullPath"`
	Size     int64  `json:"size"`
}

// MissingContentResult contains the result of a successfully completed
// find-missing-content flow.
// Its presentation state is derived from its data:
//   - TotalSourceFiles == 0: empty source
//   - len(MissingFiles) > 0: missing files
//   - otherwise: no source content is missing from the target
type MissingContentResult struct {
	TotalSourceFiles int                 `json:"totalSourceFiles"`
	MissingFiles     []MissingSourceFile `json:"missingFiles"`
	ReportPath       string              `json:"reportPath,omitempty"`
	Diagnostics      ScanDiagnostics     `json:"diagnostics"`
}

// DuplicateResult contains the result of a successfully completed duplicate flow.
type DuplicateResult struct {
	Groups      []FileHash      `json:"groups"`
	ReportPath  string          `json:"reportPath,omitempty"`
	Diagnostics ScanDiagnostics `json:"diagnostics"`
}
