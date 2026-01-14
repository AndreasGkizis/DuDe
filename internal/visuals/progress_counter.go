package visuals

import (
	"DuDe/internal/reporting"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ProgressCounter struct {
	Reporter        reporting.Reporter
	Context         context.Context
	Name            string
	senderCount     int32
	currentProgress int64
	Wg              sync.WaitGroup
	senderWg        sync.WaitGroup
	Channel         chan int
	DoneChannel     chan int
}

func NewProgressCounter(ctx context.Context, reporter reporting.Reporter, name string, senderCount int) *ProgressCounter {
	return &ProgressCounter{
		Reporter:    reporter,
		Context:     ctx,
		senderCount: int32(senderCount),
		Name:        name,
		Channel:     make(chan int),
	}
}

func (pc *ProgressCounter) SenderFinished() {
	if atomic.AddInt32(&pc.senderCount, -1) == 0 {
		close(pc.Channel)
	}
	pc.senderWg.Done()
}

func (pc *ProgressCounter) updateProgressCounterLoop(name string) {
	// Ensure pc.Wg.Done() is called when the loop exits
	defer pc.Wg.Done()

	// 1. Setup Ticker for UI Updates
	const updateInterval = 250 * time.Millisecond
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	pc.Reporter.LogProgress(
		pc.Context,
		name,
		0,
	)

	for {
		select {
		case <-pc.Context.Done():
			// CANCELLATION: Log final count and exit cleanly.
			currentCount := atomic.LoadInt64(&pc.currentProgress)
			pc.Reporter.LogDetailedStatus(pc.Context, fmt.Sprintf("Read stopped (Cancelled after %d files)", currentCount))
			return
		case <-ticker.C:
			// TICK: This is the UI Update trigger.
			currentCount := atomic.LoadInt64(&pc.currentProgress)

			// Only update the UI if the count has changed since the last tick
			pc.Reporter.LogDetailedStatus(pc.Context, fmt.Sprintf("Read %d files", currentCount))

		case _, ok := <-pc.Channel:
			if !ok {
				// Channel closed (all senders finished normally): Exit loop
				currentCount := atomic.LoadInt64(&pc.currentProgress)

				// Final log and 100% progress must be sent immediately upon completion
				pc.Reporter.LogDetailedStatus(pc.Context, fmt.Sprintf("Finished reading %d files.", currentCount))

				return
			}
			// 4. Normal increment of progress
			pc.Increment()
		}
	}
}

func (pc *ProgressCounter) Increment() {
	atomic.AddInt64(&pc.currentProgress, 1)
}

func (pc *ProgressCounter) WaitForSenders() {
	pc.senderWg.Wait()
}

func (pc *ProgressCounter) Start() {
	pc.Wg.Add(1)
	pc.senderWg.Add(int(pc.senderCount))
	go pc.updateProgressCounterLoop(pc.Name)
}
