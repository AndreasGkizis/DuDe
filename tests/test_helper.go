package tests

import (
	"DuDe/common"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupTestFiles(t *testing.T) {
	// cmd := exec.Command("bash", "-c", "../DevHelpers/create_test_files_and_folders.sh")
	bal := common.GetEntryPointDir()
	fmt.Print(bal)
	// if err := cmd.Run(); err != nil {
	// t.Fatalf("Failed to set up test files: %v", err)
	// }
}

func tearDownTestFiles(t *testing.T) {
	cmd := exec.Command("bash", "-c", "../DevHelpers/delete_test_files.sh")
	if err := cmd.Run(); err != nil {
		t.Logf("Failed to clean up test files: %v", err)
	}
}

func getResultFilePath(t *testing.T) string {
	for i := 0; ; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Get the function name from the program counter
		fn := runtime.FuncForPC(pc)
		functionName := fn.Name()
		fullpath, _ := fn.FileLine(pc)

		if fn != nil && strings.Contains(functionName, "main.init") {

			return filepath.Join(filepath.Dir(fullpath), common.ResFilename)
		}
	}
	return ""
}
