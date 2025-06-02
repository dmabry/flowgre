// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package help provides a command to display generic help information.
package help

import (
	"fmt"
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/spf13/cobra"
)

// NewCmd creates the help command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("help", "Display generic help information.")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		printGenericHelp()
	}

	return cmd.Cmd
}

// printGenericHelp prints out the top-level generic help
func printGenericHelp() {
	common.PrintHelpHeader()
	fmt.Printf("Version: %s\n", "0.4.10")
	fmt.Println()
	fmt.Println("to print more details pass '-help' after the subcommand")
	fmt.Println()
	fmt.Println("Single is used to send a given number of flows in sequence to a collector for testing.")
	fmt.Println()
	fmt.Println("Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.")
	fmt.Println()
	fmt.Println("Record is used to record flows to a file for later replay testing.")
	fmt.Println()
	fmt.Println("Replay is used to send recorded flows to a target server.")
	fmt.Println()
	fmt.Println("Proxy is used to accept flows and relay them to multiple targets")
	fmt.Println()
}