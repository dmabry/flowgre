// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package single provides a command to send a given number of flows in sequence to a collector for testing.
package single

import (
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/dmabry/flowgre/single"
	"github.com/spf13/cobra"
)

// NewCmd creates the single command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("single", "Send a given number of flows in sequence to a collector for testing.")

	server := cmd.AddStringFlag("server", "127.0.0.1", "servername or ip address of flow collector.")
	dstPort := cmd.AddIntFlag("port", 9995, "destination port used by the flow collector.")
	srcPort := cmd.AddIntFlag("src-port", 0, "source port used by the client. If 0 a Random port between 10000-15000")
	count := cmd.AddIntFlag("count", 1, "count of flows to send in sequence.")
	hexDump := cmd.AddBoolFlag("hexdump", false, "If true, do a hexdump of the packet")
	srcRange := cmd.AddStringFlag("src-range", "10.0.0.0/8", "cidr range to use for generating source IPs for flows")
	dstRange := cmd.AddStringFlag("dst-range", "10.0.0.0/8", "cidr range to use for generating destination IPs for flows")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		single.Run(*server, *dstPort, *srcPort, *count, *srcRange, *dstRange, *hexDump)
	}

	return cmd.Cmd
}