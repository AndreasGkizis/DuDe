package visuals

import (
	common "DuDe-wails/internal/common"
	"DuDe-wails/internal/models"
	"bufio"
	"fmt"
	"os"
)

func EmptyDir(path string) {
	fmt.Println("!~~ ERROR ~~!")
	fmt.Printf("The path:\"%s\" does was empty! \n", path)
	fmt.Println("!~~ ERROR ~~!")
	fmt.Println()
	os.Exit(0)

}

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
