// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package common provides shared functionality for all commands
package common

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "flowgre",
	Short: "Flowgre is a tool for generating netflow traffic for testing Netflow collectors.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}