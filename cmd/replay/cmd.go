// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package replay provides a command to send recorded flows to a target server.
package replay

import (
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/dmabry/flowgre/replay"
	"github.com/spf13/cobra"
)

// NewCmd creates the replay command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("replay", "Send recorded flows to a target server.")

	server := cmd.AddStringFlag("server", "127.0.0.1", "target server to replay flows at")
	port := cmd.AddIntFlag("port", 9995, "target server udp port")
	delay := cmd.AddIntFlag("delay", 100, "number of milliseconds between packets sent")
	db := cmd.AddStringFlag("db", "recorded_flows", "Directory to read recorded flows from")
	loop := cmd.AddBoolFlag("loop", false, "Loops the replays forever")
	workers := cmd.AddIntFlag("workers", 1, "Number of workers to spawn for replay")
	updateTS := cmd.AddBoolFlag("updatets", false, "Whether to update to the current timestamp on replayed flows")
	verbose := cmd.AddBoolFlag("verbose", false, "Whether to log every packet received. Warning can be a lot")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		replay.Run(*server, *port, *delay, *db, *loop, *workers, *updateTS, *verbose)
	}

	return cmd.Cmd
}