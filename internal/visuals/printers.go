package visuals

import (
	logger "DuDe/common"
	"DuDe/models"
	"fmt"
	"log"
	"runtime"
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
	fmt.Print(intro)
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
