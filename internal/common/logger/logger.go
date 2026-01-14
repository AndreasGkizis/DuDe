package logger

import (
	"DuDe/internal/common"
	"DuDe/internal/models"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func Initialize(enabled bool) {
	if !enabled {
		// Use a "No-Op" logger so the rest of your code doesn't crash
		// calling Logger.Info, but nothing actually happens.
		logger = zap.NewNop().Sugar()
		return
	}

	logFile, err := createLogFile()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	// Create a new encoder configuration
	encoderConfig := zap.NewProductionEncoderConfig()

	// Example 3: Use a custom time format (e.g., "2006-01-02 15:04:05")
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create a fileCore that writes logs to the file
	fileCore := zapcore.NewCore(fileEncoder, zapcore.Lock(logFile), zapcore.DebugLevel)

	// Create a core that writes logs to the console
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()) // Use console encoder for human-readable output
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.ErrorLevel)

	// Create a tee that writes to both the file and console
	teeCore := zapcore.NewTee(fileCore, consoleCore)

	// Create the logger with the core
	logger = zap.New(teeCore).Sugar()
	logger.Info("Program started!")
}

func DebugWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	logger.Debug(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func InfoWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	logger.Info(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func WarnWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	logger.Warn(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func ErrorWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	logger.Error(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func FatalWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		logger.Fatal(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	logger.Error(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func LogModelArgs(args models.ExecutionParams) {
	v := reflect.ValueOf(args)
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		logger.Info(fmt.Sprintf("Key: %s, Value: %v", fieldName, field.Interface()))
	}
}

func createLogFile() (*os.File, error) {
	var logFile *os.File
	logFilename := time.Now().Format("2006-01-02_15-04-05") + ".log"

	executableDir := common.GetExecutableDir()

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
