package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/logutils"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	logLevelTrace logutils.LogLevel = "TRACE"
	logLevelDebug logutils.LogLevel = "DEBUG"
	logLevelInfo  logutils.LogLevel = "INFO"
	logLevelWarn  logutils.LogLevel = "WARN"
	logLevelError logutils.LogLevel = "ERROR"
)

var logFilter *logutils.LevelFilter

func initLogging() {
	dir, _ := os.Getwd()

	lumberjackLogger := &lumberjack.Logger{
		Filename:   path.Join(dir, "log", "plugin.log"),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}

	log.SetOutput(lumberjackLogger)
	log.Println("lumberjack logging initialised")

	// Setup level filtering
	// TODO: it seems the level is always coming in as "OFF"
	// https://github.com/pact-foundation/pact-plugins/blob/main/drivers/rust/driver/src/plugin_manager.rs#L244
	// Hard coding to DEBUG for now
	if logFilter == nil {
		logFilter = &logutils.LevelFilter{
			Levels:   []logutils.LogLevel{logLevelTrace, logLevelDebug, logLevelInfo, logLevelWarn, logLevelError},
			MinLevel: logutils.LogLevel("DEBUG"),
			// MinLevel: logutils.LogLevel(detectLogLevel()),
			Writer: lumberjackLogger,
		}
		log.SetOutput(logFilter)
		log.Println("[DEBUG] initialised logging")
	}
}

// SetLogLevel sets the default log level for the Pact framework
func SetLogLevel(level logutils.LogLevel) error {
	switch level {
	case logLevelTrace, logLevelDebug, logLevelError, logLevelInfo, logLevelWarn:
		logFilter.SetMinLevel(level)
		return nil
	default:
		return fmt.Errorf(`invalid logLevel '%s'. Please specify one of "TRACE", "DEBUG", "INFO", "WARN", "ERROR"`, level)
	}
}

func detectLogLevel() logutils.LogLevel {
	// Log to file if specified
	var level logutils.LogLevel = "INFO"
	logLevel := logutils.LogLevel(strings.ToUpper(os.Getenv("LOG_LEVEL")))

	if logLevel != "" {
		level = logLevel
	}

	return level
}
