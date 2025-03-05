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
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
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

	// Create a new logger with development configuration or configure anything custom we like.
}

func PanicAndLog(e error) {
	if e != nil {
		Logger.DPanic(e)
	}
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

func createLogFile() (*os.File, error) {
	var logFile *os.File
	logFilename := time.Now().Format("2006-01-02_15-04-05") + ".log"

	executableDir := GetEntryPointDir()

	basedir := filepath.Join(executableDir, "logs")
	logFilepath := filepath.Join(basedir, logFilename)

	// Check if the logs directory exists
	if _, err := os.Stat(basedir); os.IsNotExist(err) {
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
