package log

import (
	"log"
	"os"
)

var verbose = false

var (
	debugLogger = log.New(os.Stderr, "", log.Lmsgprefix)
	infoLogger  = log.New(os.Stderr, "", log.Lmsgprefix)
)

func SetVerbose(b bool) {
	verbose = b
}

func Debug(s string, args ...any) {
	if verbose {
		debugLogger.Printf(s, args...)
	}
}

func Info(format string, args ...any) {
	infoLogger.Printf(format, args...)
}

func Fatalf(format string, args ...any) {
	infoLogger.Printf("FATAL: "+format, args...)
	os.Exit(1)
}

var Printf = Info
