package visuals

import (
	"DuDe/internal/reporting"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	log "DuDe/internal/common/logger"
)

type ProgressTracker struct {
	Reporter              reporting.Reporter
	Context               context.Context
	Name                  string
	BarLength             int
	totalFiles            int64
	currentProgress       int64
	lastDisplayedProgress int
	wg                    sync.WaitGroup
}

func NewProgressTracker(ctx context.Context, reporter reporting.Reporter, name string) *ProgressTracker {
	return &ProgressTracker{Reporter: reporter, Context: ctx, Name: name}
}

func (pt *ProgressTracker) updateProgressBarLoop(name string) {
	var percentage float64
	defer pt.wg.Done()

	// 1. Setup Ticker for UI Updates
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pt.Context.Done():
			// Log that the process was stopped prematurely.
			curr := atomic.LoadInt64(&pt.currentProgress)
			log.DebugWithFuncName(fmt.Sprintf("'%s' stopped due to context cancellation after processing %d files.", name, curr))
			return
		case <-ticker.C:
			curr := float64(atomic.LoadInt64(&pt.currentProgress))
			tot := float64(atomic.LoadInt64(&pt.totalFiles))

			isItTheStart := curr == 0
			if curr == 0 {
				percentage = 0
			} else {
				percentage = curr / tot * 100
				isItTheStart = false
			}
			pt.Reporter.LogProgress(pt.Context, name, float64(percentage))

			progress := int(float64(pt.BarLength) * percentage / 100)

			pt.Reporter.LogDetailedStatus(
				pt.Context,
				fmt.Sprintf("%s %.2f%%  ...%d of %d Files", name, percentage, int(curr), int(tot)),
			)

			pt.lastDisplayedProgress = progress

			if curr == tot && !isItTheStart {

				pt.Reporter.LogDetailedStatus(
					pt.Context,
					fmt.Sprintf("%s %.2f%%  ...%d of %d Files | Done.", name, percentage, int(curr), int(tot)),
				)
				return
			}
		}
	}
}

func (pt *ProgressTracker) AddTotal(count int64) {
	atomic.AddInt64(&pt.totalFiles, count)
}

func (pt *ProgressTracker) Increment() {
	atomic.AddInt64(&pt.currentProgress, 1)
}

func (pt *ProgressTracker) DecrementFromTotal() {
	atomic.AddInt64(&pt.totalFiles, -1)
}

func (pt *ProgressTracker) Wait() {
	pt.wg.Wait()
}

func (pt *ProgressTracker) Start(barLength int) {
	pt.wg.Add(1)
	pt.BarLength = barLength
	pt.lastDisplayedProgress = 0

	go pt.updateProgressBarLoop(pt.Name)
}
