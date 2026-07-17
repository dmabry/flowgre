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
func (c *ProxyCommand) Execute() error {
	targets := []string(c.targets)
	if err := config.ValidateProxy(*c.ip, *c.port, targets); err != nil {
		return fmt.Errorf("validate proxy config: %w", err)
	}
	mgr := lifecycle.New()
	defer mgr.Cancel()
	_ = mgr.SetupSignalHandler()
	if err := proxy.RunCtx(mgr.Context(), *c.ip, *c.port, *c.verbose, targets); err != nil {
		return fmt.Errorf("proxy: %w", err)
	}
	return nil
}

// RunProxy is the entry point for the proxy subcommand.
func RunProxy(args []string) {
	c := &ProxyCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	if err := c.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "proxy: %v\n", err)
		os.Exit(1)
	}
}
