package handlers

import (
	common "DuDe/common"
	process "DuDe/internal/processing"
	"DuDe/internal/static"
	"flag"
	"os"
	"strings"
)

func GetCLIArgs() map[string]string {
	result := make(map[string]string)
	mode := flag.String("mode", "", "use sf for single-folder or df for dual-folder.")

	flag.Parse()
	result["mode"] = *mode

	return result
}

func GetFileArguments(args []string) map[string]string {

	result := make(map[string]string, 0)
	basedir := "."

	targetsPath, _ := process.FindFullFilePath(basedir, static.GetArgFilename())
	dat, err := os.ReadFile(targetsPath)
	common.PanicAndLog(err)

	lines := strings.Split(string(dat), "\n")

	for _, line := range lines {

		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		for _, arg := range args {
			if key == arg {
				result[arg] = value
				break
			}
		}
	}

	return result
}
