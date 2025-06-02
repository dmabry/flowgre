// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package barrage provides a command to send a continuous barrage of flows in different sequence to a collector for testing.
package barrage

import (
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/cmd/common"
	"github.com/dmabry/flowgre/models"
	"github.com/spf13/cobra"
	"log"
	"strconv"
	"sync"
)

// NewCmd creates the barrage command
func NewCmd() *cobra.Command {
	cmd := common.NewBaseCommand("barrage", "Send a continuous barrage of flows in different sequence to a collector for testing.")

	server := cmd.AddStringFlag("server", "127.0.0.1", "servername or ip address of the flow collector")
	dstPort := cmd.AddIntFlag("port", 9995, "destination port used by the flow collector")
	srcRange := cmd.AddStringFlag("src-range", "10.0.0.0/8", "cidr range to use for generating source IPs for flows")
	dstRange := cmd.AddStringFlag("dst-range", "10.0.0.0/8", "cidr range to use for generating destination IPs for flows")
	workers := cmd.AddIntFlag("workers", 4, "number of workers to create. Unique sources per worker")
	delay := cmd.AddIntFlag("delay", 100, "number of milliseconds between packets sent")
	configFile := cmd.AddStringFlag("config", "", "Config file to use. Supersedes all given args")
	webPort := cmd.AddIntFlag("web-port", 8080, "Port to bind the web server on")
	webIP := cmd.AddStringFlag("web-ip", "0.0.0.0", "IP address the web server will listen on")
	web := cmd.AddBoolFlag("web", false, "Whether to use the web server or not")

	cmd.Cmd.Run = func(cmd *cobra.Command, args []string) {
		if *configFile != "" {
			log.Printf("Reading config file... ignoring any other given arguments\n\n")
			// TODO: Implement config file parsing with Cobra
			log.Fatalf("Config file support is not yet implemented with Cobra")
		} else {
			bConfig := models.Config{
				Server:   *server,
				DstPort:  *dstPort,
				SrcRange: *srcRange,
				DstRange: *dstRange,
				Delay:    *delay,
				Workers:  *workers,
				Web:      *web,
				WebIP:    *webIP,
				WebPort:  *webPort,
			}

			barrage.Run(&bConfig)
		}
	}

	return cmd.Cmd
}