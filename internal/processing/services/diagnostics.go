package services

import (
	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"sort"
	"sync"
	"sync/atomic"
)

// DiagnosticSeverity controls the severity used by the asynchronous logger.
type DiagnosticSeverity string

const (
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityError   DiagnosticSeverity = "error"
)

// DiagnosticEvent contains the public diagnostic plus the original error used
// for classification and internal logging.
type DiagnosticEvent struct {
	Diagnostic models.Diagnostic
	Err        error
	Severity   DiagnosticSeverity
}

// ErrorDisposition controls whether an event is retained and whether the
// operation that reported it may continue.
type DiagnosticHandler func(DiagnosticEvent) ErrorDisposition

type ErrorDisposition int

const (
	ErrorLogOnly ErrorDisposition = iota
	ErrorDiagnostic
	ErrorFatal
)

// DiagnosticSink retains accepted diagnostics and logs submitted events
// asynchronously. Retention is intentionally mutex-protected and lossless;
// only logging uses an asynchronous queue.
type DiagnosticSink struct {
	recordMu sync.Mutex
	records  []models.Diagnostic

	logMu     sync.Mutex
	logQueue  []DiagnosticEvent
	wake      chan struct{}
	stop      chan struct{}
	logDone   chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
}

// NewDiagnosticSink creates a sink for one execution.
func NewDiagnosticSink() *DiagnosticSink {
	sink := &DiagnosticSink{
		wake:    make(chan struct{}, 1),
		stop:    make(chan struct{}),
		logDone: make(chan struct{}),
	}
	go sink.logLoop()
	return sink
}

// Submit logs every event and retains only diagnostic events. It returns false
// when called after the sink has been closed.
func (s *DiagnosticSink) Submit(event DiagnosticEvent, disposition ErrorDisposition) bool {
	if s.closed.Load() {
		return false
	}

	if disposition == ErrorDiagnostic {
		s.recordMu.Lock()
		s.records = append(s.records, event.Diagnostic)
		s.recordMu.Unlock()
	}

	s.logMu.Lock()
	if s.closed.Load() {
		s.logMu.Unlock()
		return false
	}
	s.logQueue = append(s.logQueue, event)
	s.logMu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
	return true
}

// Close stops logging after all queued events have been processed.
func (s *DiagnosticSink) Close() {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		close(s.stop)
		select {
		case s.wake <- struct{}{}:
		default:
		}
	})
	<-s.logDone
}

// Snapshot returns a deterministic defensive copy of retained diagnostics.
func (s *DiagnosticSink) Snapshot() []models.Diagnostic {
	s.recordMu.Lock()
	result := append([]models.Diagnostic(nil), s.records...)
	s.recordMu.Unlock()

	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Phase != right.Phase {
			return left.Phase < right.Phase
		}
		return left.Reason < right.Reason
	})
	if result == nil {
		return []models.Diagnostic{}
	}
	return result
}

// CompletedSnapshot returns a defensive snapshot marked completed. It does not
// close the sink or determine execution success; the caller chooses this status.
func (s *DiagnosticSink) CompletedSnapshot() models.ScanDiagnostics {
	return models.ScanDiagnostics{
		Status:      models.DiagnosticStatusCompleted,
		Diagnostics: s.Snapshot(),
	}
}

func (s *DiagnosticSink) logLoop() {
	defer close(s.logDone)
	for {
		s.logMu.Lock()
		queue := s.logQueue
		s.logQueue = nil
		s.logMu.Unlock()
		for _, event := range queue {
			s.logEvent(event)
		}

		select {
		case <-s.wake:
		case <-s.stop:
			s.logMu.Lock()
			remaining := s.logQueue
			s.logQueue = nil
			s.logMu.Unlock()
			for _, event := range remaining {
				s.logEvent(event)
			}
			return
		}
	}
}

func (s *DiagnosticSink) logEvent(event DiagnosticEvent) {
	message := event.Diagnostic.Reason
	if event.Diagnostic.Path != "" {
		message = event.Diagnostic.Path + ": " + message
	}
	if event.Err != nil && event.Diagnostic.Reason == "" {
		message = event.Err.Error()
	}
	if event.Severity == DiagnosticSeverityError {
		log.ErrorWithFuncName(message)
		return
	}
	log.WarnWithFuncName(message)
}
