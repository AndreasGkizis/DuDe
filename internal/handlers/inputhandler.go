package handlers

import (
	common "DuDe/common"
	process "DuDe/internal/processing"
	"flag"
	"os"
	"strings"
)

// Loads both File and CLI Arguments.
// CLI arguments Override the file arguments.
func LoadArgs() map[string]string {

	fileArguments := []string{
		common.ArgFilename_cacheDir,
		common.ArgFilename_resDir,
		common.ArgFilename_sourceDir,
		common.ArgFilename_targetDir,
	}

	loadedFileArgs := getFileArguments(fileArguments)
	result := getCLIArgs(loadedFileArgs)
	return result
}

func getCLIArgs(result map[string]string) map[string]string {
	var curMode string
	var debugMode string
	var sourceDir string
	var targetDir string
	var cacheDir string
	var resultDir string

	flagsMap := make(map[string]string)

	flag.StringVar(&debugMode, common.DbgFlagName_long, common.Def, "activate debugger to get all kinds of logs and traces")
	flag.StringVar(&debugMode, common.DbgFlagName, common.Def, "activate debugger to get all kinds of logs and traces")

	flag.StringVar(&curMode, common.ModeFlag_long, common.Def, "use sf for single-folder or df for dual-folder.")
	flag.StringVar(&curMode, common.ModeFlag, common.Def, "use sf for single-folder or df for dual-folder.")

	flag.StringVar(&sourceDir, common.SourceFlag_long, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")
	flag.StringVar(&sourceDir, common.SourceFlag, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")

	flag.StringVar(&targetDir, common.TargetFlag_long, common.Def, "The directory of the source folder [absolute path].")
	flag.StringVar(&targetDir, common.TargetFlag, common.Def, "The directory of the source folder [absolute path].")

	flag.StringVar(&cacheDir, common.CacheDirFlag_long, common.Def, "The directory where the `memory.db` file will be kept and created [relative path].")
	flag.StringVar(&cacheDir, common.CacheDirFlag, common.Def, "The directory where the `memory.db` file will be kept and created [relative path].")

	flag.StringVar(&resultDir, common.ResultDirFlag_long, common.Def, "The directory where the `results.csv` file will be kept and created [relative path].")
	flag.StringVar(&resultDir, common.ResultDirFlag, common.Def, "The directory where the `results.csv` file will be kept and created [relative path].")

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		flagsMap[f.Name] = f.Value.String()
	})

	for key, flag := range flagsMap {
		if flag != common.Def {
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

func getFileArguments(args []string) map[string]string {

	result := make(map[string]string, 0)
	basedir := "."

	argumentsPath, _ := process.FindFullFilePath(basedir, common.ArgFilename)
	data, err := os.ReadFile(argumentsPath)
	common.PanicAndLog(err)

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		for _, arg := range args {
			if key == arg {
				if strings.HasPrefix(value, "<") {
					result[arg] = common.Def
					break
				}
				if value != common.Def {
					result[arg] = value
					break
				}
			}
		}
	}

	return result
}
