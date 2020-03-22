package logutil

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	EnableDebug = false
	EnableColor = true

	Yel   = color.New(color.FgYellow).SprintFunc()
	Green = color.New(color.FgGreen).SprintFunc()
	Red   = color.New(color.FgRed).SprintFunc()
	Bold  = color.New(color.Bold).SprintFunc()
	Gray  = color.New(color.FgHiBlack).SprintFunc()
)

// Prints to stderr.
func Debugf(format string, a ...interface{}) {
	if !EnableDebug {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: ", Gray("debug"))
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Prints to stderr.
func Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: ", Red("error"))
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Prints to stderr.
func Infof(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: ", Yel("info"))
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}
