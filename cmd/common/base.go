// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package common provides shared functionality for all commands
package common

import (
	"fmt"

	"github.com/spf13/cobra"
)

// BaseCommand represents a base command with common functionality
type BaseCommand struct {
	Cmd *cobra.Command
}

// NewBaseCommand creates a new base command
func NewBaseCommand(name, shortDesc string) *BaseCommand {
	cmd := &cobra.Command{
		Use:  name,
		Short: shortDesc,
	}

	return &BaseCommand{Cmd: cmd}
}

// Execute runs the command
func (bc *BaseCommand) Execute() error {
	return bc.Cmd.Execute()
}

// AddStringFlag adds a string flag to the command
func (bc *BaseCommand) AddStringFlag(name, defValue, usage string) *string {
	var value string
	bc.Cmd.Flags().StringVarP(&value, name, "", defValue, usage)
	return &value
}

// AddIntFlag adds an int flag to the command
func (bc *BaseCommand) AddIntFlag(name string, defValue int, usage string) *int {
	var value int
	bc.Cmd.Flags().IntVarP(&value, name, "", defValue, usage)
	return &value
}

// AddBoolFlag adds a bool flag to the command
func (bc *BaseCommand) AddBoolFlag(name string, defValue bool, usage string) *bool {
	var value bool
	bc.Cmd.Flags().BoolVarP(&value, name, "", defValue, usage)
	return &value
}

// PrintHelpHeader prints the help header for Flowgre
func PrintHelpHeader() {
	fmt.Printf("   ___ _                             \n  / __\\ | _____      ____ _ _ __ ___ \n / _\\ | |/ _" +
		" \\ \\ /\\ / / _` | '__/ _ \\\n/ /   | | (_) \\ V  V / (_| | | |  __/\n\\/    |_|\\___/ \\_/\\_/ \\__, |_|  \\" +
		"___|\n                      |___/          \n")
	fmt.Println("Slinging packets since 2022!")
	fmt.Println("Used for Netflow Collector Stress testing and other fun activities.")
}