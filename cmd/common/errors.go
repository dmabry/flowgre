// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package common provides shared functionality for all commands
package common

import (
	"log"
	"os"
)

// FatalError logs an error message and exits the program with a non-zero status
func FatalError(msg string, err error) {
	log.Printf("Fatal: %s - %v\n", msg, err)
	os.Exit(1)
}

// Error logs an error message but does not exit the program
func Error(msg string, err error) {
	if err != nil {
		log.Printf("Error: %s - %v\n", msg, err)
	}
}