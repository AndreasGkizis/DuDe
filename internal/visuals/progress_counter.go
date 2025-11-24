package visuals

import (
	"DuDe-wails/internal/reporting"
	"fmt"
	"sync"
	"sync/atomic"
)

type ProgressCounter struct {
	Reporter        reporting.Reporter
	Name            string
	Spinner         ProgressSpinner
	senderCount     int32
	currentProgress int64
	Wg              sync.WaitGroup
	senderWg        sync.WaitGroup
	Channel         chan int
	DoneChannel     chan int
}

func NewProgressCounter(name string, senderCount int, reporter reporting.Reporter) *ProgressCounter {
	return &ProgressCounter{
		Reporter:    reporter, // Store the interface implementation
		senderCount: int32(senderCount),
		Spinner:     *NewSpinner(),
		Name:        name,
		Channel:     make(chan int),
	}
}

func (pc *ProgressCounter) TotalSenders(total int32) {
	atomic.AddInt32(&pc.senderCount, total)
	pc.senderWg.Add(int(total))
}

func (pc *ProgressCounter) SenderFinished() {
	if atomic.AddInt32(&pc.senderCount, -1) == 0 {
		close(pc.Channel)
	}
	pc.senderWg.Done()
}

func (pc *ProgressCounter) updateProgressCounterLoop(name string) {

	pc.Reporter.LogProgress(
		name,
		0,
	)

	for range pc.Channel {
		pc.Spinner.Spin()
		pc.Increment()
		pc.Reporter.LogDetailedStatus(fmt.Sprintf("Read %d files", pc.currentProgress))
	}
	pc.Reporter.LogProgress(
		name,
		100,
	)
}

func (pc *ProgressCounter) Increment() {
	atomic.AddInt64(&pc.currentProgress, 1)
}

func (pc *ProgressCounter) Wait() {
	pc.senderWg.Wait()
}

func (pc *ProgressCounter) Start() {
	pc.Wg.Add(1)
	pc.senderWg.Add(int(pc.senderCount))
	go pc.updateProgressCounterLoop(pc.Name)
}
