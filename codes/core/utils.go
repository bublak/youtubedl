package core

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

// print everything
func PrintE(data interface{}) {
	fmt.Printf("%+v\n", data)
}

// LogError write to STD output + to log
func LogError(err error, str string) {
	fmt.Println(trace())

	log.Printf(str+" %s\n", err)
}

// Trace try to identify caller
func trace() (string, int, string) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "?", 0, "?"
	}

	fn := runtime.FuncForPC(pc)
	return file, line, fn.Name()
}

// FileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
