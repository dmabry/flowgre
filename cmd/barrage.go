// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"fmt"

	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/config"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/netflow"
)

// BarrageCommand holds flags and state for the barrage subcommand.
type BarrageCommand struct {
	server           *string
	port             *int
	srcRange         *string
	dstRange         *string
	workers          *int
	delay            *int
	templateInterval *int
	configFile       *string
	webPort          *int
	webIP            *string
	web              *bool
	protocol         *string
	profile          *string
}

// ParseFlags parses command-line flags for the barrage mode.
func (c *BarrageCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("barrage", flag.ExitOnError)
	c.server = fs.String("server", "127.0.0.1", "servername or ip address of the flow collector (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "destination port used by the flow collector")
	c.srcRange = fs.String("src-range", "10.0.0.0/8", "CIDR range for source IPs (IPv4 or IPv6)")
	c.dstRange = fs.String("dst-range", "10.0.0.0/8", "CIDR range for destination IPs (IPv4 or IPv6)")
	c.workers = fs.Int("workers", 4, "number of workers to create. Unique sources per worker")
	c.delay = fs.Int("delay", 100, "number of milliseconds between packets sent")
	c.templateInterval = fs.Int("template-interval", 30, "seconds between template retransmissions (0 to disable)")
	c.configFile = fs.String("config", "", "Config file to use. Supersedes all given args")
	c.webPort = fs.Int("web-port", 8080, "Port to bind the web server on")
	c.webIP = fs.String("web-ip", "0.0.0.0", "IP address the web server will listen on (IPv4 or IPv6)")
	c.web = fs.Bool("web", false, "Whether to use the web server or not")
	c.protocol = fs.String("protocol", "netflow", "protocol to use: netflow or ipfix")
	c.profile = fs.String("profile", "generic", "flow profile: generic, minimal, extended")
	return fs.Parse(args)
}

// Execute runs the barrage mode with parsed flags.
func (c *BarrageCommand) Execute() error {
	// Resolve profile from string
	var nfProfile netflow.FlowProfile
	switch *c.profile {
	case "minimal":
		nfProfile = &netflow.MinimalProfile{}
	case "extended":
		nfProfile = &netflow.ExtendedProfile{}
	default:
		nfProfile = &netflow.GenericProfile{}
	}

	// If config file is provided, load from it and ignore other args
	if *c.configFile != "" {
		fmt.Println("Reading config file... ignoring any other given arguments")
		if err := config.InitViper(*c.configFile); err != nil {
			return fmt.Errorf("error reading config file: %v", err)
		}
		cfg, err := config.LoadBarrageConfig()
		if err != nil {
			return fmt.Errorf("error loading barrage config: %v", err)
		}
		if *c.protocol == "ipfix" {
			barrage.Run(cfg, barrage.IPFIX())
		} else {
			barrage.Run(cfg, barrage.NetFlow(nfProfile))
		}
		return nil
	}

	// Run with command-line args
	cfg := &models.Config{
		Server:           *c.server,
		DstPort:          *c.port,
		SrcRange:         *c.srcRange,
		DstRange:         *c.dstRange,
		Delay:            *c.delay,
		TemplateInterval: *c.templateInterval,
		Workers:          *c.workers,
		WebIP:            *c.webIP,
		WebPort:          *c.webPort,
		Web:              *c.web,
		Protocol:         *c.protocol,
	}
	if *c.protocol == "ipfix" {
		barrage.Run(cfg, barrage.IPFIX())
	} else {
		barrage.Run(cfg, barrage.NetFlow(nfProfile))
	}
	return nil
}

// RunBarrage is the entry point for the barrage subcommand.
func RunBarrage(args []string) error {
	c := &BarrageCommand{}
	if err := c.ParseFlags(args); err != nil {
		return err
	}
	return c.Execute()
}
