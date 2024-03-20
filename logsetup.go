package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"time"
)

func setGlobalLogLevel(logLevel string) error {
	logLevels := map[string]zerolog.Level{
		"debug":   zerolog.DebugLevel,
		"warning": zerolog.WarnLevel,
		"info":    zerolog.InfoLevel,
		"error":   zerolog.ErrorLevel,
	}

	if level, ok := logLevels[logLevel]; ok {
		zerolog.SetGlobalLevel(level)
	} else {
		return fmt.Errorf("undefined log level: %v", logLevel)
	}

	return nil
}

func SetupZeroLog(logLevel string) {
	// zerolog.levelFieldName =
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if err := setGlobalLogLevel("debug"); err != nil {
		fmt.Printf("Failed to set global log level: %s", err.Error())
	}
}
