package visuals

import (
	common "DuDe/internal/common"
	"DuDe/internal/models"
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	fmt.Println()
	fmt.Printf("\nSeems like this is the first time you run DuDe, welcome!")
	fmt.Printf("\nThe '%s' file was not found! So a NEW one has been created for you =].\n", common.ArgFilename)
	fmt.Print("Follow these steps:\n")
	fmt.Printf("1. Open the newly created '%s' file.\n", common.ArgFilename)
	fmt.Print("2. Add the paths you want to the folders you want to scan.\n")
	fmt.Print("3. Save the file.\n")
	fmt.Print("4. Run the program again.\n")

	waitAndExit()
}

func DefaultSource() {
	fmt.Printf("\nThe source directory indicated seems to be the default one ... Duuuuuude...you can't do that man")
}

func EnterToExit() {
	fmt.Println()
	fmt.Println("--------> Press ENTER key to exit <--------")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	os.Exit(0)
}

func waitAndExit() {
	fmt.Println()
	fmt.Println("Dude!")
	fmt.Println()
	fmt.Println("--------> The program will now stop <--------")
	fmt.Println()
	EnterToExit()
}

func Intro() {
	fmt.Print(common.CLI_Intro)
}
func Outro() {
	fmt.Println()
	fmt.Println("Duuuuuuuuuuude, all Done!")
	fmt.Println()
	fmt.Println("Thank you for using this program")
	fmt.Println("...Made by A.G with <3...")
	EnterToExit()
}

func FirstRun(args models.ExecutionParams) {
	if args.SourceDir == common.Def {
		ArgsFileNotFound()
	} else {
		ComparingFolders(args)
	}
}

func ComparingFolders(args models.ExecutionParams) {
	sourceDir := args.SourceDir
	targetDir := args.TargetDir

	if targetDir != common.Def && targetDir != "" {
		fmt.Printf("\nComparing files in: %s\n", sourceDir)
		fmt.Printf("\nWith files in: %s\n", targetDir)
	} else {
		fmt.Printf("\nLooking for duplicates in: %s\n", sourceDir)
	}
}

func NoDuplicatesFound() {
	fmt.Println("No duplicates were found")
}

type ProgressTracker struct {
	Name                  string
	BarLength             int
	Spinner               ProgressSpinner
	totalFiles            int64
	currentProgress       int64
	lastDisplayedProgress int
	wg                    sync.WaitGroup
}

func NewProgressTracker(name string) *ProgressTracker {
	return &ProgressTracker{Spinner: *NewSpinner(), Name: name}
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

			progress := int(float64(pt.BarLength) * percentage / 100)
			progressBar := strings.Repeat("█", progress) + strings.Repeat("░", pt.BarLength-progress)

			pt.Spinner.Spin()
			fmt.Printf("\r%s: %s %.2f%% %s  ...%d of %d Files", name, progressBar, percentage, pt.Spinner.Print(), int(curr), int(tot))
			pt.lastDisplayedProgress = progress

			if curr == tot && !isItTheStart {
				fmt.Printf("\r%s: %s %.2f%%    ...%d of %d Files | Done.", name, progressBar, percentage, int(curr), int(tot))
				fmt.Println()
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

func (pc *ProgressCounter) updateProgressCounterLoop2(name string) {
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.Spinner.Spin()
			fmt.Printf("\r%s: %s  ...%d Files", name, pc.Spinner.Print(), int(pc.currentProgress))

		case _, ok := <-pc.Channel:
			if !ok {
				// Channel closed, print final status and return
				fmt.Printf("\r%s: Done   %d Files\n", name, int(pc.currentProgress))
				return
			}
			pc.Increment()
			// Optionally print here if you want immediate feedback on increment
			// but spinner will update on next tick anyway
		}
	}
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

type ProgressSpinner struct {
	States       []string
	CurrentState int
}

func NewSpinner() *ProgressSpinner {
	return &ProgressSpinner{
		States: []string{"-", "\\", "|", "/"},
	}
}

func (sp *ProgressSpinner) Spin() {
	if sp.CurrentState+1 >= len(sp.States) {
		sp.CurrentState = 0
	} else {
		sp.CurrentState++
	}
}

func (sp *ProgressSpinner) Start() {
	sp.CurrentState = 0
}
func (sp *ProgressSpinner) Print() string {
	return sp.States[sp.CurrentState]
}
