// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"os"

	"github.com/dmabry/flowgre/proxy"
)

// targetFlags is a custom flag.Value for parsing multiple --target flags.
type targetFlags []string

func (f *targetFlags) String() string {
	return "<multiple>"
}

func (f *targetFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// ProxyCommand holds flags and state for the proxy subcommand.
type ProxyCommand struct {
	ip      *string
	port    *int
	targets targetFlags
	verbose *bool
}

// ParseFlags parses command-line flags for the proxy mode.
func (c *ProxyCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("proxy", flag.ExitOnError)
	c.ip = fs.String("ip", "127.0.0.1", "IP address proxy should listen on (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "proxy listen udp port")
	fs.Var(&c.targets, "target", "Can be passed multiple times in IP:PORT format")
	c.verbose = fs.Bool("verbose", false, "Whether to log every flow received. Warning can be a lot")
	return fs.Parse(args)
}

// Execute runs the proxy mode with parsed flags.
func (c *ProxyCommand) Execute() {
	proxy.Run(*c.ip, *c.port, *c.verbose, []string(c.targets))
}

// RunProxy is the entry point for the proxy subcommand.
func RunProxy(args []string) {
	c := &ProxyCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	c.Execute()
}
