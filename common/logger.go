package logger

import (
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

// init func always runs first no matter what. its a go thing
func init() {
	// Create a new logger with development configuration or configure anything custom we like.
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	logger = l.Sugar()
}

func GetLogger() *zap.SugaredLogger {
	return logger
}

func PanicAndLog(e error) {
	if e != nil {
		logger.DPanic(e)
	}
}
