package visuals

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type ProgressCounter struct {
	Name            string
	Spinner         ProgressSpinner
	senderCount     int32
	currentProgress int64
	Wg              sync.WaitGroup
	senderWg        sync.WaitGroup
	Channel         chan int
	DoneChannel     chan int
}

func NewProgressCounter(name string, senderCount int) *ProgressCounter {
	return &ProgressCounter{
		senderCount: int32(senderCount),
		Spinner:     *NewSpinner(), Name: name,
		Channel: make(chan int),
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

	fmt.Printf("\r%s: %s  ...%d Files", name, pc.Spinner.Print(), int(pc.currentProgress))

	for range pc.Channel {
		pc.Spinner.Spin()
		pc.Increment()
		fmt.Printf("\r%s: %s  ...%d Files", name, pc.Spinner.Print(), int(pc.currentProgress))
	}
	fmt.Printf("\r%s: Done   %d Files", name, int(pc.currentProgress))
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
