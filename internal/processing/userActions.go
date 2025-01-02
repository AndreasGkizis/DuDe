package processing

import (
	"bufio"
	"fmt"
	"os"
)

func WaitAndExit() {
	fmt.Println("\nPress Enter to exit...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	os.Exit(0)
}
