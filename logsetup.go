package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	level  string
	logger zerolog.Logger
}

type TSWriter struct {
	consoleWriter zerolog.ConsoleWriter
	mu            sync.Mutex
}

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

func (w *TSWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.consoleWriter.Write(p)
}

func (w *TSWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.consoleWriter.Close()
}

func setLogLevel(logLevel string) error {
	if level, exists := logLevelsMap[logLevel]; exists {
		zerolog.SetGlobalLevel(level)
	} else {
		return fmt.Errorf("undefined log level: %v", logLevel)
	}
	return nil
}

// func SetupZeroLog(logLevel string) {
// 	zerolog.TimeFieldFormat = time.RFC822
// 	logLevel = strings.ToLower(logLevel)

// 	if err := setLogLevel(logLevel); err != nil {
// 		fmt.Printf("Failed to set global log level: %s", err.Error())
// 	} else {
// 		setLogLevel("debug")
// 	}
// }

func NewLogger(logLevel string) *Logger {
	logLevel = strings.ToLower(logLevel)

	if err := setLogLevel(logLevel); err != nil {
		fmt.Printf("Failed to set global log level: %s", err.Error())
	} else {
		setLogLevel("debug")
	}

	// ConsoleWriter is not thread-safe, so we have to make a wrapper around it
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC822}
	output.FormatLevel = func(l interface{}) string {
		return strings.ToUpper(fmt.Sprintf("|%s|", l))
	}
	output.FormatFieldName = func(name interface{}) string {
		return fmt.Sprintf("%s: ", name)
	}
	output.FormatMessage = func(msg interface{}) string {
		return fmt.Sprintf("Msg: %s", msg)
	}

	l := Logger{
		level:  logLevel,
		logger: zerolog.New(&TSWriter{consoleWriter: output}).With().Timestamp().Logger(),
	}

	return &l
}
