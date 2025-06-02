// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package record provides a command to record flows to a file for later replay testing.
package record

import (
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/dmabry/flowgre/record"
	"github.com/spf13/cobra"
)

// NewCmd creates the record command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("record", "Record flows to a file for later replay testing.")

	ip := cmd.AddStringFlag("ip", "127.0.0.1", "ip address record should listen on")
	port := cmd.AddIntFlag("port", 9995, "listen udp port")
	db := cmd.AddStringFlag("db", "recorded_flows", "Directory to place recorded flows for later replay")
	verbose := cmd.AddBoolFlag("verbose", false, "Whether to log every packet received. Warning can be a lot")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		record.Run(*ip, *port, *db, *verbose)
	}

	return cmd.Cmd
}