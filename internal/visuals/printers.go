package visuals

import (
	common "DuDe/common"
	"DuDe/models"
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

func DirDoesNotExistMessage(path string) {
	fmt.Printf("ERROR !... The path \"%s\" does not exist... ! ERROR\n", path)
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
	fmt.Println("--------> Press Enter key to exit <--------")
	fmt.Println()
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	os.Exit(0)
}

func PrintDuplicates(input []models.DuDeFile) {

	logger := common.GetLogger()
	for _, file := range input {
		if len(file.DuplicatesFound) > 0 {
			logger.Infof("File: %s, Duplicates: %d", file.Filename, len(file.DuplicatesFound))
			for _, dup := range file.DuplicatesFound {
				logger.Infof("\tDuplicate: %s", dup.Filename)
			}
		}
	}
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

func MonitorGoroutines(stopChan chan struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond) // Adjust interval as needed
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			log.Println("Stopping goroutine.")
		case <-ticker.C:
			// Print the current number of goroutines
			log.Printf("Active goroutines: %d\n", runtime.NumGoroutine())
		}
	}
}

func MonitorProgress(totalFiles int, progressCh <-chan int) {

	var currentProgress int
	var percentage float64

	for {
		select {
		case currentProgress = <-progressCh:

			percentage = float64(currentProgress) / float64(totalFiles) * 100

			barLength := 50 // Length of the progress bar in characters
			progress := int(float64(barLength) * percentage / 100)
			progressBar := strings.Repeat("█", progress) + strings.Repeat("░", barLength-progress)

			fmt.Printf("\rProgress: %s %.1f %%", progressBar, percentage)

			if currentProgress == totalFiles {
				fmt.Println("\nAll files processed!")
				return
			}
		}
	}
}
