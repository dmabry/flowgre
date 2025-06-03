// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package common provides shared functionality for all commands
package common

import (
	"log"
	"os"
)

// Logger is a wrapper around the standard log package with additional functionality
type Logger struct {
	*log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(prefix string) *Logger {
	return &Logger{log.New(os.Stdout, prefix+": ", log.Ldate|log.Ltime|log.Lshortfile)}}
