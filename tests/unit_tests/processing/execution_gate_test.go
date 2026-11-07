package unit_tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"DuDe/internal/processing"
)

func TestExecutionGateResetCancelsAndWaitsForActiveExecution(t *testing.T) {
	gate := processing.NewExecutionGate()
	executionContext, err := gate.Start(context.Background())
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}

	resetDone := make(chan bool, 1)
	go func() {
		resetDone <- gate.BeginResetAndWait()
	}()

	select {
	case <-executionContext.Done():
	case <-time.After(time.Second):
		t.Fatal("reset did not cancel the active execution")
	}

	select {
	case <-resetDone:
		t.Fatal("reset completed before the active execution finished")
	case <-time.After(25 * time.Millisecond):
	}

	gate.Finish()
	select {
	case ownsReset := <-resetDone:
		if !ownsReset {
			t.Fatal("expected the first reset caller to own reset cleanup")
		}
	case <-time.After(time.Second):
		t.Fatal("reset did not continue after execution cleanup finished")
	}
	gate.EndReset()
}

func TestExecutionGateRejectsStartsDuringReset(t *testing.T) {
	gate := processing.NewExecutionGate()
	if !gate.BeginResetAndWait() {
		t.Fatal("expected first reset caller to own reset cleanup")
	}
	defer gate.EndReset()

	_, err := gate.Start(context.Background())
	if !errors.Is(err, processing.ErrResetInProgress) {
		t.Fatalf("expected reset-in-progress error, got %v", err)
	}
}

func TestExecutionGateConcurrentResetWaitsForOwner(t *testing.T) {
	gate := processing.NewExecutionGate()
	if !gate.BeginResetAndWait() {
		t.Fatal("expected first reset caller to own reset cleanup")
	}

	waiterDone := make(chan bool, 1)
	go func() {
		waiterDone <- gate.BeginResetAndWait()
	}()

	select {
	case <-waiterDone:
		t.Fatal("concurrent reset returned before the owner finished")
	case <-time.After(25 * time.Millisecond):
	}

	gate.EndReset()
	select {
	case ownsReset := <-waiterDone:
		if ownsReset {
			t.Fatal("expected concurrent reset caller not to repeat reset cleanup")
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent reset did not return after owner finished")
	}
}
