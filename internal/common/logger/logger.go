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
	fileCore := zapcore.NewCore(fileEncoder, zapcore.Lock(logFile), zapcore.DebugLevel)

	// Create a core that writes logs to the console
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()) // Use console encoder for human-readable output
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.ErrorLevel)

	// Create a tee that writes to both the file and console
	teeCore := zapcore.NewTee(fileCore, consoleCore)

	// Create the logger with the core
	Logger = zap.New(teeCore).Sugar()
	Logger.Info("Program started!")
}

func DebugWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		Logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Debug(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func InfoWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		Logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Info(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func WarnWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		Logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Warn(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func ErrorWithFuncName(message string) {
	pc, _, lineNum, ok := runtime.Caller(1)
	if !ok {
		Logger.Error(fmt.Sprintf("Could not get caller info: %s", message)) // Log a warning without the function name
		return
	}
	funcName := runtime.FuncForPC(pc).Name()

	Logger.Error(fmt.Sprintf("%s()(line:%d)-> [%s]", funcName, lineNum, message))
}

func LogArgs(args map[string]string) {
	for key, value := range args {
		Logger.Info(fmt.Sprintf("Key: %s, Value: %s", key, value))
	}
}

func LogModelArgs(args models.ExecutionParams) {
	v := reflect.ValueOf(args)
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		Logger.Info(fmt.Sprintf("Key: %s, Value: %v", fieldName, field.Interface()))
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
