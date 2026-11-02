package models

// ExecutionMode identifies which operation an execution request represents.
// Its zero value is intentionally invalid: every request must select a mode.
type ExecutionMode uint8

const (
	ExecutionModeDuplicates         ExecutionMode = 1
	ExecutionModeFindMissingContent ExecutionMode = 2
)

type ExecutionParams struct {
	Mode                 ExecutionMode `json:"mode"`
	Directories          []string      `json:"directories"`
	FindMissingSourceDir string        `json:"findMissingSourceDir"`
	FindMissingTargetDir string        `json:"findMissingTargetDir"`
	UseCache             bool          `json:"useCache"`
	CacheDir             string        `json:"cacheDir"`
	ResultsDir           string        `json:"resultsDir"`
	ParanoidMode         bool          `json:"paranoidMode"`
	WorkerCount          int           `json:"workerCount"`
	BufferSizeBytes      int           `json:"bufferSizeBytes"`
	DebugMode            bool          `json:"debugMode"`
}
