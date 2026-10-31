// Package flows contains the execution-flow contract, registry, and concrete
// operations available to the processing lifecycle.
package flows

import (
	"DuDe/internal/models"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"fmt"
)

var (
	ErrExecutionModeUnavailable       = errors.New("execution mode is not connected yet")
	ErrNilExecutionFlow               = errors.New("execution flow cannot be nil")
	ErrExecutionFlowAlreadyRegistered = errors.New("execution flow is already registered for this mode")
	ErrExecutionModeRequired          = errors.New("execution mode must be provided")
	ErrUnknownExecutionMode           = errors.New("unknown execution mode")
)

// ResultPublisher publishes concrete successful flow results. It deliberately
// has one method per result type instead of introducing a shared result model.
type ResultPublisher interface {
	PublishDuplicateResult(result models.DuplicateResult)
	PublishMissingContentResult(result models.MissingContentResult)
}

// ExecutionFlow is one executable operation supported by the application.
type ExecutionFlow interface {
	Mode() models.ExecutionMode
	Validate(args *models.ExecutionParams) error
	Run(ctx context.Context, args models.ExecutionParams, reporter reporting.Reporter, publisher ResultPublisher) error
}

// Registry stores the executable flow registered for each mode.
type Registry struct {
	flows map[models.ExecutionMode]ExecutionFlow
}

// NewRegistry creates a registry containing the supplied flows.
func NewRegistry(registeredFlows ...ExecutionFlow) (*Registry, error) {
	registry := &Registry{
		flows: make(map[models.ExecutionMode]ExecutionFlow, len(registeredFlows)),
	}

	for _, flow := range registeredFlows {
		if err := registry.Register(flow); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// Register connects a flow to the mode returned by that flow.
func (r *Registry) Register(flow ExecutionFlow) error {
	if flow == nil {
		return ErrNilExecutionFlow
	}
	if r.flows == nil {
		r.flows = make(map[models.ExecutionMode]ExecutionFlow)
	}

	mode := flow.Mode()
	if !isKnownExecutionMode(mode) {
		return fmt.Errorf("execution mode %d: %w", mode, ErrUnknownExecutionMode)
	}
	if _, exists := r.flows[mode]; exists {
		return fmt.Errorf("execution mode %d: %w", mode, ErrExecutionFlowAlreadyRegistered)
	}

	r.flows[mode] = flow
	return nil
}

// Resolve returns the registered flow for an explicit, recognized mode.
func (r *Registry) Resolve(mode models.ExecutionMode) (ExecutionFlow, error) {
	if mode == 0 {
		return nil, fmt.Errorf("Mode: %w", ErrExecutionModeRequired)
	}
	if !isKnownExecutionMode(mode) {
		return nil, fmt.Errorf("Mode (%d): %w", mode, ErrUnknownExecutionMode)
	}

	flow, exists := r.flows[mode]
	if !exists {
		return nil, fmt.Errorf("execution mode %d: %w", mode, ErrExecutionModeUnavailable)
	}

	return flow, nil
}

func isKnownExecutionMode(mode models.ExecutionMode) bool {
	switch mode {
	case models.ExecutionModeDuplicates, models.ExecutionModeFindMissingContent:
		return true
	default:
		return false
	}
}
