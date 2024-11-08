package logs

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()

	lumberjackLogger := &lumberjack.Logger{
		Filename:   "./logs/appointment.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	mutliwriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	logger.SetOutput(mutliwriter)
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
}
