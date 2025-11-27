package visuals

import (
	"DuDe/internal/reporting"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ProgressTracker struct {
	Reporter              reporting.Reporter
	Name                  string
	BarLength             int
	Spinner               ProgressSpinner
	totalFiles            int64
	currentProgress       int64
	lastDisplayedProgress int
	wg                    sync.WaitGroup
}

func NewProgressTracker(name string, reporter reporting.Reporter) *ProgressTracker {
	return &ProgressTracker{Spinner: *NewSpinner(), Name: name, Reporter: reporter}
}

func (pt *ProgressTracker) updateProgressBarLoop(name string) {
	var percentage float64
	defer pt.wg.Done()
	ticker := time.NewTicker(150 * time.Millisecond) // Adjust the interval as needed
	defer ticker.Stop()

	for {
		select {
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
			pt.Reporter.LogProgress(name, int(curr))

			progress := int(float64(pt.BarLength) * percentage / 100)

			pt.Spinner.Spin()
			pt.Reporter.LogDetailedStatus(
				fmt.Sprintf("%s %.2f%% %s  ...%d of %d Files", name, percentage, pt.Spinner.Print(), int(curr), int(tot)),
			)

			pt.lastDisplayedProgress = progress

			if curr == tot && !isItTheStart {

				pt.Reporter.LogDetailedStatus(
					fmt.Sprintf("%s %.2f%% %s  ...%d of %d Files | Done.", name, percentage, pt.Spinner.Print(), int(curr), int(tot)),
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
	fmt.Println()

	go pt.updateProgressBarLoop(pt.Name)
}
