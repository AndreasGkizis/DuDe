package models

// DiagnosticPhase identifies the operation in which a diagnostic occurred.
type DiagnosticPhase string

const (
	DiagnosticPhaseWalking DiagnosticPhase = "walking"
	DiagnosticPhaseHashing DiagnosticPhase = "hashing"
)

// DiagnosticScope identifies the logical scan collection involved.
type DiagnosticScope string

const (
	DiagnosticScopeSource    DiagnosticScope = "source"
	DiagnosticScopeTarget    DiagnosticScope = "target"
	DiagnosticScopeDuplicate DiagnosticScope = "duplicate"
)

// DiagnosticStatus describes the lifecycle state represented by diagnostics.
type DiagnosticStatus string

const (
	DiagnosticStatusNotStarted   DiagnosticStatus = "notStarted"
	DiagnosticStatusRunning      DiagnosticStatus = "running"
	DiagnosticStatusCompleted    DiagnosticStatus = "completed"
	DiagnosticStatusCancelled    DiagnosticStatus = "cancelled"
	DiagnosticStatusInconclusive DiagnosticStatus = "inconclusive"
)

// Diagnostic records a path-specific, non-fatal scan failure.
type Diagnostic struct {
	Path   string          `json:"path"`
	Reason string          `json:"reason"`
	Phase  DiagnosticPhase `json:"phase"`
	Scope  DiagnosticScope `json:"scope"`
}

// ScanDiagnostics is the current execution's diagnostic snapshot and state.
type ScanDiagnostics struct {
	Status      DiagnosticStatus `json:"status"`
	Diagnostics []Diagnostic     `json:"diagnostics"`
}

// NewScanDiagnostics creates diagnostics with a non-nil empty collection.
func NewScanDiagnostics(status DiagnosticStatus) ScanDiagnostics {
	return ScanDiagnostics{Status: status, Diagnostics: []Diagnostic{}}
}
