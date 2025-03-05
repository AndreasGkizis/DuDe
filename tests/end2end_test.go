package tests

import (
	"os"
	"os/exec"
	"testing"
)

func Test_Should_find_duplicates(t *testing.T) {
	defer tearDownTestFiles(t)
	setupTestFiles(t)

	// Run the application
	cmd := exec.Command("go", "run", "../cmd/main.go")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Application failed to run: %v\nOutput: %s", err, string(output))
	}
	// Run your duplicate finder function
	results := getResultFilePath(t)

	lines, _ := os.ReadFile(results)

	if lines == nil {
		t.Errorf("the results file is empty")
	}

}
