package processing

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrExecutionRunning = errors.New("execution already running")
	ErrResetInProgress  = errors.New("reset already in progress")
)

type ExecutionGate struct {
	mu              sync.Mutex
	running         bool
	resetting       bool
	cancelExecution context.CancelFunc
	executionDone   chan struct{}
	resetDone       chan struct{}
}

func NewExecutionGate() *ExecutionGate {
	return &ExecutionGate{}
}

func (gate *ExecutionGate) Start(parent context.Context) (context.Context, error) {
	gate.mu.Lock()
	defer gate.mu.Unlock()

	if gate.resetting {
		return nil, ErrResetInProgress
	}
	if gate.running {
		return nil, ErrExecutionRunning
	}

	ctx, cancel := context.WithCancel(parent)
	gate.running = true
	gate.cancelExecution = cancel
	gate.executionDone = make(chan struct{})
	return ctx, nil
}

func (gate *ExecutionGate) Cancel() {
	gate.mu.Lock()
	cancel := gate.cancelExecution
	gate.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (gate *ExecutionGate) Finish() {
	gate.mu.Lock()
	if !gate.running {
		gate.mu.Unlock()
		return
	}

	cancel := gate.cancelExecution
	done := gate.executionDone
	gate.cancelExecution = nil
	gate.executionDone = nil
	gate.running = false
	close(done)
	gate.mu.Unlock()

	cancel()
}

func (gate *ExecutionGate) BeginResetAndWait() bool {
	gate.mu.Lock()
	if gate.resetting {
		resetDone := gate.resetDone
		gate.mu.Unlock()
		<-resetDone
		return false
	}

	gate.resetting = true
	gate.resetDone = make(chan struct{})
	cancel := gate.cancelExecution
	executionDone := gate.executionDone
	gate.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if executionDone != nil {
		<-executionDone
	}
	return true
}

func (gate *ExecutionGate) EndReset() {
	gate.mu.Lock()
	if !gate.resetting {
		gate.mu.Unlock()
		return
	}

	resetDone := gate.resetDone
	gate.resetDone = nil
	gate.resetting = false
	close(resetDone)
	gate.mu.Unlock()
}
