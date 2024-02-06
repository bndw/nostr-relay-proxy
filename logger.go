package main

import (
	"log"
	"os"
)

const (
	logLevelDebug logLevel = iota
	logLevelError
)

type logLevel int

func parseLogLevel(s string) logLevel {
	switch s {
	case "debug":
		return logLevelDebug
	default:
		return logLevelError
	}
}

type logger interface {
	Infof(format string, v ...any)
	Warningf(format string, v ...any)
	Errorf(format string, v ...any)
	setLogLevel(level logLevel)
}

func newLogger() logger {
	l := log.New(os.Stderr, "", log.LstdFlags)
	return &stdLogger{
		log:      l,
		logLevel: logLevelError,
	}
}

type stdLogger struct {
	log      *log.Logger
	logLevel logLevel
}

func (l *stdLogger) setLogLevel(level logLevel) {
	l.logLevel = level
}

func (l stdLogger) Infof(format string, v ...any) {
	if l.logLevel != logLevelDebug {
		return
	}
	l.log.Printf(format, v...)
}

func (l stdLogger) Warningf(format string, v ...any) {
	if l.logLevel != logLevelDebug {
		return
	}
	l.log.Printf(format, v...)
}

func (l stdLogger) Errorf(format string, v ...any) {
	l.log.Printf(format, v...)
}
