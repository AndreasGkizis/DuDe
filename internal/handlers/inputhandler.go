package handlers

import (
	common "DuDe/common"
	"flag"
	"os"
	"path/filepath"
	"strings"
)

func LoadArgs() map[string]string {

	args := make(map[string]string)
	args[common.ArgFilename_cacheDir] = common.Def
	args[common.ArgFilename_resDir] = common.Def
	args[common.ArgFilename_sourceDir] = common.Def
	args[common.ArgFilename_targetDir] = common.Def
	args[common.ArgFilename_Dbg] = common.Def
	args[common.ArgFilename_Mode] = common.Def

	loadedFileArgs := getFileArguments(args)
	result := getCLIArgs(loadedFileArgs)
	applyDefaults(result)
	return result
}

func applyDefaults(result map[string]string) {

	executablePath, err := os.Executable()

	if err != nil {
		common.PanicAndLog(err)
	}
	executableDir := filepath.Dir(executablePath)

	if result[common.ArgFilename_cacheDir] == common.Def {
		result[common.ArgFilename_cacheDir] = filepath.Join(executableDir, common.MemFilename)
	}

	if result[common.ArgFilename_Dbg] == common.Def {
		result[common.ArgFilename_Dbg] = common.DbgFlagNotActiveValue
	}

	if result[common.ArgFilename_Mode] == common.Def {
		result[common.ArgFilename_Mode] = common.ModeSingleFolder
	}

	if result[common.ArgFilename_resDir] == common.Def {
		result[common.ArgFilename_resDir] = filepath.Join(executableDir, common.ResFilename)
	}

	if result[common.ArgFilename_sourceDir] == common.Def {
		common.Logger.Fatalf("Source Path can not be left empty, please update your %s with the correct path", common.ArgFilename)
	}

	if result[common.ArgFilename_targetDir] == common.Def {
		if result[common.ModeFlag] == common.ModeDualFolder {
			common.Logger.Fatalf("Target path can not be left empty when in Dual folder mode, please update your %s with the correct path", common.ArgFilename)
		}
	}

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

	flag.StringVar(&curMode, common.ModeFlag_long, common.Def, "use "+common.ModeSingleFolder+" for single-folder or "+common.ModeDualFolder+" for dual-folder.")
	flag.StringVar(&curMode, common.ModeFlag, common.Def, "use sf for single-folder or df for dual-folder.")

	flag.StringVar(&sourceDir, common.SourceFlag_long, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")
	flag.StringVar(&sourceDir, common.SourceFlag, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")

	flag.StringVar(&targetDir, common.TargetFlag_long, common.Def, "The directory of the source folder [absolute path].")
	flag.StringVar(&targetDir, common.TargetFlag, common.Def, "The directory of the source folder [absolute path].")

	flag.StringVar(&cacheDir, common.MemDirFlag_long, common.Def, "The directory where the "+common.MemFilename+" file will be kept and created [relative path].")
	flag.StringVar(&cacheDir, common.MemDirFlag, common.Def, "The directory where the "+common.MemFilename+" file will be kept and created [relative path].")

	flag.StringVar(&resultDir, common.ResultDirFlag_long, common.Def, "The directory where the "+common.ResFilename+" file will be created [relative path].")
	flag.StringVar(&resultDir, common.ResultDirFlag, common.Def, "The directory where the "+common.ResFilename+" file will be created [relative path].")

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		flagsMap[f.Name] = f.Value.String()
	})

	for key, flag := range flagsMap {
		if flag != common.Def {
			switch key {
			case common.DbgFlagName_long, common.DbgFlagName:
				if flag == common.DbgFlagActiveValue {
					result[key] = flag
				}
			case common.ModeFlag_long, common.ModeFlag:
				result[key] = flag
			case common.SourceFlag_long, common.SourceFlag:
				result[common.ArgFilename_sourceDir] = flag
			case common.TargetFlag_long, common.TargetFlag:
				result[common.ArgFilename_targetDir] = flag
			case common.MemDirFlag_long, common.MemDirFlag:
				result[common.ArgFilename_cacheDir] = flag
			case common.ResultDirFlag_long, common.ResultDirFlag:
				result[common.ArgFilename_resDir] = flag
			}
		}
	}
	return result
}

func getFileArguments(args map[string]string) map[string]string {

	executablePath, err := os.Executable()

	if err != nil {
		common.Logger.Fatal(err)
	}

	argumentsPath := filepath.Join(filepath.Dir(executablePath), common.ArgFilename)
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
					args[arg] = common.Def
					break
				}
				if value != common.Def {
					args[arg] = value
					break
				}
			}
		}
	}

	return args
}
