package handlers

import (
	common "DuDe/common"
	process "DuDe/internal/processing"
	"flag"
	"os"
	"strings"
)

const def = "default"

// get CLI arguments and overwrites the previous arguments
func GetCLIArgs(result map[string]string) map[string]string {
	var curMode string
	var debugMode string
	var sourceDir string
	var targetDir string
	var cacheDir string
	var resultDir string

	flagsMap := make(map[string]string)

	flag.StringVar(&debugMode, common.DbgFlagName_long, def, "activate debugger to get all kinds of logs and traces")
	flag.StringVar(&debugMode, common.DbgFlagName, def, "activate debugger to get all kinds of logs and traces")

	flag.StringVar(&curMode, "mode", def, "use sf for single-folder or df for dual-folder.")
	flag.StringVar(&curMode, "m", def, "use sf for single-folder or df for dual-folder.")

	flag.StringVar(&sourceDir, "source", def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")
	flag.StringVar(&sourceDir, "s", def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")

	flag.StringVar(&targetDir, "target", def, "The directory of the source folder [absolute path].")
	flag.StringVar(&targetDir, "t", def, "The directory of the source folder [absolute path].")

	flag.StringVar(&cacheDir, "cache-dir", def, "The directory where the `memory.csv` file will be kept and created [relative path].")
	flag.StringVar(&cacheDir, "c", def, "The directory where the `memory.csv` file will be kept and created [relative path].")

	flag.StringVar(&resultDir, "results", def, "The directory where the `results.csv` file will be kept and created [relative path].")
	flag.StringVar(&resultDir, "r", def, "The directory where the `results.csv` file will be kept and created [relative path].")

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		flagsMap[f.Name] = f.Value.String()
	})

	for key, flag := range flagsMap {
		if flag != def {
			switch key {
			case "debug", "dbg":
				if flag == common.DbgFlagActiveValue {
					result[key] = flag
				}
			case "mode", "m":
				result[key] = flag
			case "source", "s":
				result[common.ArgFilename_sourceDir] = flag
			case "target", "t":
				result[common.ArgFilename_targetDir] = flag
			case "cache-dir", "c":
				result[common.ArgFilename_cacheDir] = flag
			case "results", "r":
				result[common.ArgFilename_resDir] = flag
			}
		}
	}
	return result
}

func GetFileArguments(args []string) map[string]string {

	result := make(map[string]string, 0)
	basedir := "."

	targetsPath, _ := process.FindFullFilePath(basedir, common.ArgFilename)
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
