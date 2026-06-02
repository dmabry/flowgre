// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"os"

	"github.com/dmabry/flowgre/replay"
)

// ReplayCommand holds flags and state for the replay subcommand.
type ReplayCommand struct {
	server   *string
	port     *int
	delay    *int
	dbDir    *string
	loop     *bool
	workers  *int
	updateTS *bool
	verbose  *bool
}

// ParseFlags parses command-line flags for the replay mode.
func (c *ReplayCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	c.server = fs.String("server", "127.0.0.1", "target server to replay flows at (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "target server udp port")
	c.delay = fs.Int("delay", 100, "number of milliseconds between packets sent")
	c.dbDir = fs.String("db", "recorded_flows", "Directory to read recorded flows from")
	c.loop = fs.Bool("loop", false, "Loops the replays forever")
	c.workers = fs.Int("workers", 1, "Number of workers to spawn for replay")
	c.updateTS = fs.Bool("updatets", false, "Whether to update to the current timestamp on replayed flows")
	c.verbose = fs.Bool("verbose", false, "Whether to log every packet received. Warning can be a lot")
	return fs.Parse(args)
}

// Execute runs the replay mode with parsed flags.
func (c *ReplayCommand) Execute() {
	replay.Run(*c.server, *c.port, *c.delay, *c.dbDir, *c.loop, *c.workers, *c.updateTS, *c.verbose)
}

// RunReplay is the entry point for the replay subcommand.
func RunReplay(args []string) {
	c := &ReplayCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	c.Execute()
}
