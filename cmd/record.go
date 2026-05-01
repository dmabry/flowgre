// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"os"

	"github.com/dmabry/flowgre/record"
)

// RecordCommand holds flags and state for the record subcommand.
type RecordCommand struct {
	ip      *string
	port    *int
	dbDir   *string
	verbose *bool
}

// ParseFlags parses command-line flags for the record mode.
func (c *RecordCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	c.ip = fs.String("ip", "127.0.0.1", "ip address record should listen on")
	c.port = fs.Int("port", 9995, "listen udp port")
	c.dbDir = fs.String("db", "recorded_flows", "Directory to place recorded flows for later replay")
	c.verbose = fs.Bool("verbose", false, "Whether to log every packet received. Warning can be a lot")
	return fs.Parse(args)
}

// Execute runs the record mode with parsed flags.
func (c *RecordCommand) Execute() {
	record.Run(*c.ip, *c.port, *c.dbDir, *c.verbose)
}

// RunRecord is the entry point for the record subcommand.
func RunRecord(args []string) {
	c := &RecordCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	c.Execute()
}
