package main

import (
	"log"
	"os"
	"path"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func initLogging() {
	dir, _ := os.Getwd()
	log.SetOutput(&lumberjack.Logger{
		Filename:   path.Join(dir, "log", "plugin.log"),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
}
