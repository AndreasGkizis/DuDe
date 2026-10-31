package services

import (
	"DuDe/internal/reporting"
	"DuDe/internal/visuals"
	"context"
	"errors"
	"sync"
)

// Discovery collects eligible files from a set of directory roots. The
// underlying walk remains the same as the duplicate flow's existing walk so
// this extraction does not change filesystem eligibility or error handling.
type Discovery struct{}

// NewDiscovery creates the shared file-discovery component.
func NewDiscovery() Discovery {
	return Discovery{}
}

// Discover walks each root concurrently and returns the collected file
// metadata. Directory-walk errors are reported by WalkDir as before; only
// cancellation is returned to the caller.
func (Discovery) Discover(ctx context.Context, roots []string, reporter reporting.Reporter) (*sync.Map, error) {
	files := &sync.Map{}
	if len(roots) == 0 {
		return files, nil
	}

	reading := visuals.NewProgressCounter(ctx, reporter, "Reading", len(roots))
	reading.Start()
	for _, root := range roots {
		go WalkDir(ctx, root, files, reading)
	}
	reading.Wait()

	if err := ctx.Err(); errors.Is(err, context.Canceled) {
		return files, err
	}
	return files, nil
}
