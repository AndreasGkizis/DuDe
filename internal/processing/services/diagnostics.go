package services

import (
	log "DuDe/internal/common/logger"
	"DuDe/internal/models"
	"sort"
	"sync"
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
// asynchronously. One mutex protects admission, retention, and queueing;
// logging runs outside the lock. Submit and Close may be called concurrently.
type DiagnosticSink struct {
	mu       sync.Mutex
	records  []models.Diagnostic
	logQueue []DiagnosticEvent
	closed   bool

	wake      chan struct{}
	stop      chan struct{}
	logDone   chan struct{}
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

// Submit queues every accepted event for logging and retains diagnostic events.
// It returns false without modifying the sink if closure has begun. Acceptance
// is decided under the same lock as Close, even when they run concurrently.
func (s *DiagnosticSink) Submit(event DiagnosticEvent, disposition ErrorDisposition) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	if disposition == ErrorDiagnostic {
		s.records = append(s.records, event.Diagnostic)
	}
	s.logQueue = append(s.logQueue, event)
	s.mu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
	return true
}

// Close rejects further submissions and waits for all accepted events to be
// logged. It is safe to call concurrently or repeatedly. Snapshots remain stable
// after Close returns.
func (s *DiagnosticSink) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.stop)
		s.mu.Unlock()
	})
	<-s.logDone
}

// Snapshot returns a deterministic defensive copy of retained diagnostics.
func (s *DiagnosticSink) Snapshot() []models.Diagnostic {
	s.mu.Lock()
	result := append([]models.Diagnostic(nil), s.records...)
	s.mu.Unlock()

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
		s.mu.Lock()
		queue := s.logQueue
		s.logQueue = nil
		closed := s.closed
		s.mu.Unlock()
		for _, event := range queue {
			s.logEvent(event)
		}
		if closed {
			return
		}

		select {
		case <-s.wake:
		case <-s.stop:
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
