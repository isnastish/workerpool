package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"strings"
	"time"
)

var logLevelsMap = map[string]zerolog.Level{
	"debug":    zerolog.DebugLevel,
	"info":     zerolog.InfoLevel,
	"warning":  zerolog.WarnLevel,
	"error":    zerolog.ErrorLevel,
	"fatal":    zerolog.FatalLevel,
	"panic":    zerolog.PanicLevel,
	"disabled": zerolog.Disabled,
	"trace":    zerolog.TraceLevel,
}

func setLogLevel(logLevel string) error {
	if level, exists := logLevelsMap[strings.ToLower(logLevel)]; exists {
		zerolog.SetGlobalLevel(level)
	} else {
		return fmt.Errorf("undefined log level: %v", logLevel)
	}
	return nil
}

func SetupZeroLog(logLevel string) {
	zerolog.TimeFieldFormat = time.RFC822

	if err := setLogLevel("debug"); err != nil {
		fmt.Printf("Failed to set global log level: %s", err.Error())
		return
	}
}
