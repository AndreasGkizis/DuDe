package handlers

import (
	common "DuDe/common"
	process "DuDe/internal/processing"
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

func GetFileArguments() []string {

	result := make([]string, 0)
	basedir := "."

	targetsPath, _ := process.FindFullFilePath(basedir, "args.txt")
	dat, err := os.ReadFile(targetsPath)
	common.PanicAndLog(err)
	targets := strings.Split(string(dat), "\n")

	for _, v := range targets {
		if strings.HasPrefix(v, "source") {
			result = append(result, strings.Trim(strings.SplitAfter(v, "=")[1], "\r"))
		} else if strings.HasPrefix(v, "target") {
			result = append(result, strings.Trim(strings.SplitAfter(v, "=")[1], "\r"))
		}
	}
	// _, targetError := os.ReadDir(result["target"])
	// check(targetError)
	// _, sourceErr := os.ReadDir(result["source"])
	// check(sourceErr)

	return result
}
