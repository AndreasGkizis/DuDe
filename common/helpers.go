package common

import (
	"path/filepath"
	"runtime"
	"strings"
)

func GetEntryPointDir() string {
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
			return filepath.Dir(fullpath)
		}
	}
	return ""
}
