// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/dmabry/flowgre/config"
	"github.com/dmabry/flowgre/lifecycle"
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
	c.ip = fs.String("ip", "127.0.0.1", "IP address to listen on (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "listen udp port")
	c.dbDir = fs.String("db", "recorded_flows", "Directory to place recorded flows for later replay")
	c.verbose = fs.Bool("verbose", false, "Whether to log every packet received. Warning can be a lot")
	return fs.Parse(args)
}

// Execute runs the record mode with parsed flags.
func (c *RecordCommand) Execute() error {
	if err := config.ValidateRecord(*c.ip, *c.port, *c.dbDir); err != nil {
		return fmt.Errorf("validate record config: %w", err)
	}
	mgr := lifecycle.New()
	_ = mgr.SetupSignalHandler()
	defer mgr.Cancel()
	if err := record.RunCtx(mgr.Context(), *c.ip, *c.port, *c.dbDir, *c.verbose); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	return nil
}

// RunRecord is the entry point for the record subcommand.
func RunRecord(args []string) {
	c := &RecordCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	if err := c.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "record: %v\n", err)
		os.Exit(1)
	}
}
