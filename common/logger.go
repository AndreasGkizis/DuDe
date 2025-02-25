package common

import (
	"os"
	"path/filepath"
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
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Create a core that writes logs to the file
	core := zapcore.NewCore(encoder, zapcore.AddSync(logFile), zapcore.DebugLevel)

	// Create the logger with the core
	Logger = zap.New(core).Sugar()

	// Create a new logger with development configuration or configure anything custom we like.
}

func PanicAndLog(e error) {
	if e != nil {
		Logger.DPanic(e)
	}
}

func createLogFile() (*os.File, error) {
	logFilename := time.Now().Format("2006-01-02_15-04-05") + ".log"
	basedir := "./logs"
	var logFile *os.File
	logFilepath := filepath.Join(basedir, logFilename)

	// Check if the logs directory exists
	if _, err := os.Stat(basedir); os.IsNotExist(err) {
		// Create logs directory if it doesn't exist
		err = os.Mkdir(basedir, 0755)
		if err != nil {
			return nil, err // Return error if unable to create directory
		}
	}

	// Now try to create or open the log file
	logFile, err := os.OpenFile(logFilepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err // Return error if unable to open/create log file
	}

	return logFile, nil
}
