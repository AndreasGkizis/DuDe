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
	fmt.Printf("\nSeems like this is the first time you run DuDe, welcome!")
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

func Intro() {
	fmt.Print(common.CLI_Intro)
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
		fmt.Printf("Comparing files in: %s\n", sourceDir)
		fmt.Printf("With files in: %s\n", targetDir)
	} else {
		fmt.Printf("Looking for duplicates in: %s\n", sourceDir)
	}
}

type ProgressTracker struct {
	BarLength       int
	totalFiles      int64
	currentProgress int64
	wg              sync.WaitGroup
}

func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{}
}

func (pt *ProgressTracker) updateProgressBarloop() {
	var percentage float64
	pt.wg.Add(1)
	for {
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

		fmt.Printf("\rProgress: %s %.2f %% (...%d of %d Files)", progressBar, percentage, int(curr), int(tot))

		if curr == tot && !isItTheStart {
			fmt.Println("\nAll files processed!")
			pt.wg.Done()
			return
		}

		time.Sleep(100 * time.Millisecond) // Check every 100 milliseconds
	}
}

func (pt *ProgressTracker) AddTotal(count int64) {
	atomic.AddInt64(&pt.totalFiles, count)
}

func (pt *ProgressTracker) Increment() {
	atomic.AddInt64(&pt.currentProgress, 1)
}

func (pt *ProgressTracker) Start(barLength int) {
	pt.wg.Add(1)
	pt.BarLength = barLength
	go pt.updateProgressBarloop()
}
