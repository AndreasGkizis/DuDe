package unit_tests

import (
	"DuDe/internal/models"
	"DuDe/internal/processing/flows"
	"DuDe/internal/reporting"
	"context"
	"errors"
	"testing"
)

type registryTestFlow struct {
	mode models.ExecutionMode
}

func (flow registryTestFlow) Mode() models.ExecutionMode {
	return flow.mode
}

func (registryTestFlow) Validate(*models.ExecutionParams) error {
	return nil
}

func (registryTestFlow) Run(context.Context, models.ExecutionParams, reporting.Reporter, flows.ResultPublisher) error {
	return nil
}

func TestExecutionFlowRegistryResolvesRegisteredFlow(t *testing.T) {
	registered := registryTestFlow{mode: models.ExecutionModeDuplicates}
	registry, err := flows.NewRegistry(registered)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	resolved, err := registry.Resolve(models.ExecutionModeDuplicates)
	if err != nil {
		t.Fatalf("resolve duplicate flow: %v", err)
	}
	if resolved.Mode() != models.ExecutionModeDuplicates {
		t.Fatalf("expected duplicate mode, got %d", resolved.Mode())
	}
}

func TestExecutionFlowRegistryRequiresExplicitMode(t *testing.T) {
	registry, err := flows.NewRegistry()
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	_, err = registry.Resolve(0)
	if !errors.Is(err, flows.ErrExecutionModeRequired) {
		t.Fatalf("expected ErrExecutionModeRequired, got %v", err)
	}
}

func TestExecutionFlowRegistryRejectsUnknownMode(t *testing.T) {
	registry, err := flows.NewRegistry()
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	_, err = registry.Resolve(models.ExecutionMode(99))
	if !errors.Is(err, flows.ErrUnknownExecutionMode) {
		t.Fatalf("expected ErrUnknownExecutionMode, got %v", err)
	}
}

func TestExecutionFlowRegistryLeavesFindMissingContentDisconnected(t *testing.T) {
	registry, err := flows.NewRegistry(
		registryTestFlow{mode: models.ExecutionModeDuplicates},
	)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	_, err = registry.Resolve(models.ExecutionModeFindMissingContent)
	if !errors.Is(err, flows.ErrExecutionModeUnavailable) {
		t.Fatalf("expected ErrExecutionModeUnavailable, got %v", err)
	}
}

func TestExecutionFlowRegistryRejectsDuplicateRegistration(t *testing.T) {
	registry, err := flows.NewRegistry(
		registryTestFlow{mode: models.ExecutionModeDuplicates},
	)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	err = registry.Register(registryTestFlow{mode: models.ExecutionModeDuplicates})
	if !errors.Is(err, flows.ErrExecutionFlowAlreadyRegistered) {
		t.Fatalf("expected ErrExecutionFlowAlreadyRegistered, got %v", err)
	}
}
