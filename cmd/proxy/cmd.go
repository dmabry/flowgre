// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package proxy provides a command to accept flows and relay them to multiple targets.
package proxy

import (
	"fmt"
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/dmabry/flowgre/proxy"
	"github.com/spf13/cobra"
)

// targetFlags is used to allow for multiple targets to be passed for proxy
type targetFlags []string

// String is used to return a string form of targets passed to proxy
func (i *targetFlags) String() string {
	var output string
	var target string
	first := true

	for _, target = range *i {
		if first {
			output = target
			first = false
		} else {
			output = output + ", " + target
		}
	}
	return output
}

// Set is used to put multiple targets into a slice
func (i *targetFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// NewCmd creates the proxy command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("proxy", "Accept flows and relay them to multiple targets.")

	var proxyTargetsFlags targetFlags

	ip := cmd.AddStringFlag("ip", "127.0.0.1", "ip address proxy should listen on")
	port := cmd.AddIntFlag("port", 9995, "proxy listen udp port")
	cmd.Cmd.Flags().Var(&proxyTargetsFlags, "target", "Can be passed multiple times in IP:PORT format")
	verbose := cmd.AddBoolFlag("verbose", false, "Whether to log every flow received. Warning can be a lot")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(proxyTargetsFlags) == 0 {
			fmt.Println("Error: At least one target is required")
			return
		}
		proxy.Run(*ip, *port, *verbose, proxyTargetsFlags)
	}

	return cmd.Cmd
}