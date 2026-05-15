// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"os"

	"github.com/dmabry/flowgre/single"
)

// SingleCommand holds flags and state for the single subcommand.
type SingleCommand struct {
	server   *string
	port     *int
	srcPort  *int
	count    *int
	hexDump  *bool
	srcRange *string
	dstRange *string
}

// ParseFlags parses command-line flags for the single mode.
func (c *SingleCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("single", flag.ExitOnError)
	c.server = fs.String("server", "127.0.0.1", "servername or ip address of flow collector (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "destination port used by the flow collector.")
	c.srcPort = fs.Int("src-port", 0, "source port used by the client. If 0 a Random port between 10000-15000")
	c.count = fs.Int("count", 1, "count of flow to send in sequence.")
	c.hexDump = fs.Bool("hexdump", false, "If true, do a hexdump of the packet")
	c.srcRange = fs.String("src-range", "10.0.0.0/8", "CIDR range for source IPs (IPv4 or IPv6)")
	c.dstRange = fs.String("dst-range", "10.0.0.0/8", "CIDR range for destination IPs (IPv4 or IPv6)")
	return fs.Parse(args)
}

// Execute runs the single mode with parsed flags.
func (c *SingleCommand) Execute() {
	single.Run(*c.server, *c.port, *c.srcPort, *c.count, *c.srcRange, *c.dstRange, *c.hexDump)
}

// RunSingle is the entry point for the single subcommand.
func RunSingle(args []string) {
	c := &SingleCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	c.Execute()
}
