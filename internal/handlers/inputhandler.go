package handlers

import (
	common "DuDe/internal/common"
	"DuDe/internal/processing"
	"flag"
	"fmt"
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

	argspath := processing.CreateArgsFile()
	loadedFileArgs, _ := getFileArguments(argspath, args)
	result := getCLIArgs(loadedFileArgs)
	applyDefaults(result)

	common.LogArgs(result)

	return result
}

func applyDefaults(result map[string]string) {

	executableDir := common.GetExecutableDir()

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
		common.Logger.Fatalf("You need to enter at least a Source folder! please edit %s with a valid path, save it and run again", common.ArgFilename)
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
					result[common.ArgFilename_Dbg] = flag
				}
			case common.ModeFlag_long, common.ModeFlag:
				result[common.ArgFilename_Mode] = flag
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

func getFileArguments(path string, args map[string]string) (map[string]string, error) {

	data, err := os.ReadFile(path)

	if err != nil {
		common.Logger.DPanic(err)
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])

		value := parts[1]

		for argkey := range args {
			if key == argkey {
				updated, err := validateAndUpdatePath(value, argkey, args)
				if updated && err == nil {
					break
				} else if err != nil {
					common.Logger.Fatal(err)
					return nil, err
				}
			}

		}
	}

	return args, nil
}

func validateAndUpdatePath(value string, argkey string, args map[string]string) (bool, error) {
	value = sanitizeInput(value)
	// Check if the path is valid
	if _, err := os.Stat(value); err == nil {
		args[argkey] = value
		return true, nil
	} else if (argkey == common.ArgFilename_targetDir && value == common.Def) ||
		(argkey == common.ArgFilename_resDir && value == common.Def) {
		return true, nil
	} else {
		return false, fmt.Errorf("the selected Path: %s is invalid! Please edit to a valid path and rerun", value)
	}
}

func sanitizeInput(input string) string {
	return strings.TrimSpace(
		strings.TrimPrefix(
			strings.TrimSuffix(input, common.Path_suffix),
			common.Path_prefix),
	)
}
