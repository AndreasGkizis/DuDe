package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

// init func always runs first no matter what. its a go thing
func init() {

	logFile, err := createLogFile()
	if err != nil {
		panic(err)
	}

	// Create a new encoder configuration
	encoderConfig := zap.NewProductionEncoderConfig()

	// Example 3: Use a custom time format (e.g., "2006-01-02 15:04:05")
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create a fileCore that writes logs to the file
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zapcore.DebugLevel)

	// Create a core that writes logs to the console
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()) // Use console encoder for human-readable output
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	// Create a tee that writes to both the file and console
	teeCore := zapcore.NewTee(fileCore, consoleCore)

	// Create the logger with the core
	Logger = zap.New(teeCore).Sugar()
	Logger.Info("Program started!")
}

func DebugWithFuncName(message string) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		Logger.Debug(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Debug(fmt.Sprintf("%s() -> %s", funcName, message))
}

func WarnWithFuncName(message string) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		Logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Warn(fmt.Sprintf("%s() -> %s", funcName, message))
}

func LogArgs(args map[string]string) {
	for key, value := range args {
		Logger.Info(fmt.Sprintf("Key: %s, Value: %s", key, value))
	}
}

func createLogFile() (*os.File, error) {
	var logFile *os.File
	logFilename := time.Now().Format("2006-01-02_15-04-05") + ".log"

	executableDir := GetExecutableDir()

	basedir := filepath.Join(executableDir, "logs")
	logFilepath := filepath.Join(basedir, logFilename)

	// Check if the logs directory exists
	if _, err := os.Stat(basedir); os.IsNotExist(err) {
		fmt.Println("BELOW!!")
		fmt.Println(basedir)
		fmt.Println("ABOVE!!")
		err = os.Mkdir(basedir, 0755)
		if err != nil {
			return nil, err
		}
	}

	logFile, err := os.OpenFile(logFilepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}
