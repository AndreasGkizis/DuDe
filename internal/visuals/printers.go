package visuals

import (
	logger "DuDe/common"
	"DuDe/models"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"
)

func PrintDuplicates(input []models.DuDeFile) {

	logger := logger.GetLogger()
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
