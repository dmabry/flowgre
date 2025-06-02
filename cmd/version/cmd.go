// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package version provides a command to display version information.
package version

import (
	"fmt"
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/spf13/cobra"
)

// NewCmd creates the version command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("version", "Display version information.")

	const version = "0.4.10" // semantic version
	const license = "Apache License, Version 2.0"

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		common.PrintHelpHeader()
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("License: %s\n", license)
	}

	return cmd.Cmd
}