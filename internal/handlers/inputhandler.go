package handlers

import (
	common "DuDe-wails/internal/common"
	"DuDe-wails/internal/common/logger"
	"DuDe-wails/internal/models"
	"DuDe-wails/internal/processing"
	"DuDe-wails/internal/visuals"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func LoadArgs() models.ExecutionParams {

	args := make(map[string]string)

	args[common.ArgFilename_cacheDir] = common.Def
	args[common.ArgFilename_resDir] = common.Def
	args[common.ArgFilename_sourceDir] = common.Def
	args[common.ArgFilename_targetDir] = common.Def
	args[common.ArgFilename_paranoidMode] = common.Def

	argsPath := processing.CreateArgsFile()
	loadedFileArgs, _ := GetFileArguments(argsPath, args)
	finalArgs := GetCLIArgs(loadedFileArgs)
	applyDefaults(finalArgs)

	results := convertToObject(args)

	logger.LogModelArgs(results)

	return results
}

func convertToObject(args map[string]string) models.ExecutionParams {

	params := models.ExecutionParams{

		SourceDir:  args[common.ArgFilename_sourceDir],
		TargetDir:  args[common.ArgFilename_targetDir],
		CacheDir:   args[common.ArgFilename_cacheDir],
		ResultsDir: args[common.ArgFilename_resDir],
		CPUs:       runtime.NumCPU(), // decide defaults here
		BufSize:    500,
	}
	value, err := strconv.ParseBool(args[common.ArgFilename_paranoidMode])

	if err != nil {
		logger.ErrorWithFuncName(fmt.Sprintf("Error Parsing string to boolean: %v", err))
	}

	params.ParanoidMode = value
	params.DualFolderModeEnabled = params.IsDualFolderMode()
	return params
}

func applyDefaults(result map[string]string) {

	executableDir := common.GetExecutableDir()

	if result[common.ArgFilename_cacheDir] == common.Def {
		result[common.ArgFilename_cacheDir] = filepath.Join(executableDir, common.MemFilename)
	}

	if result[common.ArgFilename_resDir] == common.Def {
		result[common.ArgFilename_resDir] = filepath.Join(executableDir, common.ResFilename)
	}

	if result[common.ArgFilename_paranoidMode] == common.Def {
		result[common.ArgFilename_paranoidMode] = "false"
	}

	if result[common.ArgFilename_sourceDir] == common.Def {
		logger.Logger.Errorf(`The %s was set to the default value ("%s").`, common.ArgFilename_sourceDir, common.Def)
		visuals.DefaultSource()
		visuals.ArgsFileNotFound()
	}
}

func GetCLIArgs(result map[string]string) map[string]string {
	var sourceDir string
	var targetDir string
	var cacheDir string
	var resultDir string
	var enableParanoid bool

	flagsMap := make(map[string]string)

	flag.StringVar(&sourceDir, common.SourceFlag_long, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")
	flag.StringVar(&sourceDir, common.SourceFlag, common.Def, "The directory of the source folder [absolute path](also the only folder in single folder mode).")

	flag.StringVar(&targetDir, common.TargetFlag_long, common.Def, "The directory of the source folder [absolute path].")
	flag.StringVar(&targetDir, common.TargetFlag, common.Def, "The directory of the source folder [absolute path].")

	flag.StringVar(&cacheDir, common.MemDirFlag_long, common.Def, "The directory where the "+common.MemFilename+" file will be kept and created [relative path].")
	flag.StringVar(&cacheDir, common.MemDirFlag, common.Def, "The directory where the "+common.MemFilename+" file will be kept and created [relative path].")

	flag.StringVar(&resultDir, common.ResultDirFlag_long, common.Def, "The directory where the "+common.ResFilename+" file will be created [relative path].")
	flag.StringVar(&resultDir, common.ResultDirFlag, common.Def, "The directory where the "+common.ResFilename+" file will be created [relative path].")

	flag.BoolVar(&enableParanoid, common.ParanoidFlag, false, "Enable super duplicate checking.")
	flag.BoolVar(&enableParanoid, common.ParanoidFlag_long, false, "Enable super duplicate checking.")

	flag.Parse()

	if flag.NFlag() == 0 {
		return result
	}

	flag.Visit(func(f *flag.Flag) {
		flagsMap[f.Name] = f.Value.String()
	})

	for key, flag := range flagsMap {
		if flag != common.Def {
			switch key {
			case common.SourceFlag_long, common.SourceFlag:
				result[common.ArgFilename_sourceDir] = flag
			case common.TargetFlag_long, common.TargetFlag:
				result[common.ArgFilename_targetDir] = flag
			case common.MemDirFlag_long, common.MemDirFlag:
				result[common.ArgFilename_cacheDir] = flag
			case common.ResultDirFlag_long, common.ResultDirFlag:
				result[common.ArgFilename_resDir] = flag
			case common.ParanoidFlag_long, common.ParanoidFlag:
				result[common.ArgFilename_paranoidMode] = flag
			}
		}
	}
	return result
}

func GetFileArguments(path string, args map[string]string) (map[string]string, error) {

	data := common.Must(os.ReadFile(path))

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])

		value := parts[1]

		for argKey := range args {
			if key == argKey {
				updated, err := validateAndUpdatePath(value, argKey, args)
				if updated && err == nil {
					break
				} else if err != nil {
					logger.ErrorWithFuncName(err.Error())
					return nil, err
				}
			}
		}
	}

	return args, nil
}

func validateAndUpdatePath(value string, argKey string, args map[string]string) (bool, error) {
	value = SanitizeInput(value)
	// Check if the path is valid
	if _, err := os.Stat(value); err == nil {
		args[argKey] = value
		return true, nil
	} else if value == common.ArgFilename_sourceDir_example ||
		value == common.ArgFilename_targetDir_example ||
		value == common.ArgFilename_resDir_example ||
		value == "" {
		args[argKey] = common.Def
		return true, nil
	} else if (argKey == common.ArgFilename_targetDir && value == common.Def) ||
		(argKey == common.ArgFilename_resDir && value == common.Def) {
		return true, nil
	} else {
		return false, err
	}
}

func SanitizeInput(input string) string {
	return strings.TrimSpace(
		strings.TrimPrefix(
			strings.TrimSuffix(input, common.Path_suffix),
			common.Path_prefix),
	)
}
