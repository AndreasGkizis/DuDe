package common

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

// init func always runs first no matter what. its a go thing
func init() {

	logFile, err := createLogFile()
	if err != nil {
		panic(err)
	}

	// Create a new encoder configuration
	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create a core that writes logs to the file
	core := zapcore.NewCore(encoder, zapcore.AddSync(logFile), zapcore.DebugLevel)

	// Create the logger with the core
	logger = zap.New(core).Sugar()

	// Create a new logger with development configuration or configure anything custom we like.
}

func GetLogger() *zap.SugaredLogger {
	return logger
}

func PanicAndLog(e error) {
	if e != nil {
		logger.DPanic(e)
	}
}

func createLogFile() (*os.File, error) {
	logFilename := time.Now().Format("20060102_150405") + ".log"
	basedir := "./logs"
	var logFile *os.File
	logFilepath := filepath.Join(basedir, logFilename)

	// logFile, err := os.OpenFile(logFilepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	_, err := os.Stat(logFilepath)
	if os.IsNotExist(err) {
		logFile, err := os.Create(logFilepath)

		if err != nil {
			panic(err)
		}

		return logFile, nil
	}

	if err != nil {
		panic(err)
	} else {
		logFile, _ = os.Open(logFilepath)
	}
	return logFile, nil
}
