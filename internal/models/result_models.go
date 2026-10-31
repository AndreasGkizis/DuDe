package models

// MissingSourceFile describes one source file that has no equivalent target file.
type MissingSourceFile struct {
	Filename string `json:"filename"`
	FullPath string `json:"fullPath"`
	// maybe more, we will see what the UI needs to display for each missing file
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
}

// DuplicateResult contains the result of a successfully completed duplicate flow.
type DuplicateResult struct {
	Groups     []FileHash `json:"groups"`
	ReportPath string     `json:"reportPath,omitempty"`
}
