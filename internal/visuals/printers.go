package visuals

import (
	common "DuDe/common"
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

func DirDoesNotExistMessage(path string) {
	fmt.Println("!~~ ERROR ~~!")
	fmt.Printf("The path:\"%s\" does not exist\n", path)
	fmt.Println("!~~ ERROR ~~!")
	fmt.Println()
	fmt.Println("!~~ How to solve this issue ~~!")
	fmt.Println()
	fmt.Println()
	fmt.Println("1. Open the Arguments.txt and make sure the paths there are valid")
	fmt.Println("2. Correct paths if needed and make sure the file is saved.")
	fmt.Println("3. Try running the program again")

	waitAndExit()
}

func ArgsFileNotFound() {
	fmt.Printf("\nThe '%s' file was not found! So a NEW one has been created for you =].\n", common.ArgFilename)
	fmt.Print("Follow these steps:\n")
	fmt.Printf("1. Open the newly created '%s' file.\n", common.ArgFilename)
	fmt.Print("2. Add the paths you want to the folders you want to scan.\n")
	fmt.Print("3. Save the file.\n")
	fmt.Print("4. Run the program again.\n")

	waitAndExit()
}

func waitAndExit() {
	fmt.Println()
	fmt.Println()
	fmt.Println("--------> Press ENTER key to exit <--------")
	fmt.Println()
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	os.Exit(0)
}

func PrintIntro() {
	intro := `
 ██████╗  ██╗   ██╗        ██████╗  ███████╗ 
 ██╔══██╗ ██║   ██║        ██╔══██╗ ██╔════╝ 
 ██║  ██║ ██║   ██║ █████╗ ██║  ██║ █████╗   
 ██║  ██║ ██║   ██║ ╚════╝ ██║  ██║ ██╔══╝   
 ██████╔╝ ╚██████╔╝        ██████╔╝ ███████╗ 
 ╚═════╝   ╚═════╝         ╚═════╝  ╚══════╝ 
 --------------------------------------------
 Welcome to Duplicate Detection CLI         
 --------------------------------------------
 
 🔍 Let's find those duplicates...  
 💀 ..and....KILL 'EM!`
	fmt.Print(intro + "\n")
}

func (pt *ProgressTracker) printProgressBar() {
	var percentage float64
	percentage = float64(atomic.LoadInt64(&pt.currentProgress)) / float64(atomic.LoadInt64(&pt.totalFiles)) * 100

	progress := int(float64(pt.BarLength) * percentage / 100)
	progressBar := strings.Repeat("█", progress) + strings.Repeat("░", pt.BarLength-progress)

	fmt.Printf("\rProgress: %s %.1f %%", progressBar, percentage)

	if atomic.LoadInt64(&pt.currentProgress) == atomic.LoadInt64(&pt.totalFiles) {
		fmt.Println("\nAll files processed!")
		return
	}
}

func (pt *ProgressTracker) updateProgressBar() {

	var percentage float64
	for range pt.ProgressChan {
		percentage = float64(atomic.LoadInt64(&pt.currentProgress)) / float64(atomic.LoadInt64(&pt.totalFiles)) * 100

		progress := int(float64(pt.BarLength) * percentage / 100)
		progressBar := strings.Repeat("█", progress) + strings.Repeat("░", pt.BarLength-progress)

		pt.mu.Lock()
		fmt.Printf("\rProgress: %s %.1f %%", progressBar, percentage)
		pt.mu.Unlock()

		if atomic.LoadInt64(&pt.currentProgress) == atomic.LoadInt64(&pt.totalFiles) {
			fmt.Println("\nAll files processed!")
			pt.wg.Done()
			return
		}
	}
}

func (pt *ProgressTracker) updateProgressBar2() {

	percentage := float64(atomic.LoadInt64(&pt.currentProgress)) / float64(atomic.LoadInt64(&pt.totalFiles)) * 100

	progress := int(float64(pt.BarLength) * percentage / 100)
	progressBar := strings.Repeat("█", progress) + strings.Repeat("░", pt.BarLength-progress)

	pt.mu.Lock()
	fmt.Printf("\rProgress: %s %.1f %%", progressBar, percentage)
	pt.mu.Unlock()

	if atomic.LoadInt64(&pt.currentProgress) == atomic.LoadInt64(&pt.totalFiles) {
		fmt.Println("\nAll files processed!")
		pt.wg.Done()
		return
	}
}

type ProgressTracker struct {
	ProgressChan    chan int
	BarLength       int
	totalFiles      int64
	currentProgress int64
	mu              sync.Mutex
	wg              sync.WaitGroup
}

func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{ProgressChan: make(chan int, 100)}
}

func (pt *ProgressTracker) AddTotal(count int64) {
	atomic.AddInt64(&pt.totalFiles, count)
}

func (pt *ProgressTracker) Increment() {
	atomic.AddInt64(&pt.currentProgress, 1)
	pt.printProgressBar()
}

func (pt *ProgressTracker) Start(barLength int) {
	pt.wg.Add(1)
	pt.BarLength = barLength
	// go pt.updateProgressBar()
}
