// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/config"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/web"
	"golang.org/x/crypto/bcrypt"
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
	webUsername      *string
	webPassword      *string
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
	c.webIP = fs.String("web-ip", "127.0.0.1", "IP address the web server will listen on (IPv4 or IPv6)")
	c.web = fs.Bool("web", false, "Whether to use the web server or not")
	c.protocol = fs.String("protocol", "netflow", "protocol to use: netflow or ipfix")
	c.profile = fs.String("profile", "generic", "flow profile: generic, minimal, extended")
	c.webUsername = fs.String("web-username", "", "Web server username (default: env FLOWGRE_WEB_USERNAME or generated)")
	c.webPassword = fs.String("web-password", "", "Web server password (default: env FLOWGRE_WEB_PASSWORD or generated)")
	return fs.Parse(args)
}

// resolveProfile returns the FlowProfile for the given profile string.
func resolveProfile(profile string) netflow.FlowProfile {
	switch profile {
	case "minimal":
		return &netflow.MinimalProfile{}
	case "extended":
		return &netflow.ExtendedProfile{}
	default:
		return &netflow.GenericProfile{}
	}
}

// validateProtocol returns an error if the protocol is not supported.
func validateProtocol(protocol string) error {
	switch protocol {
	case "netflow", "ipfix":
		return nil
	default:
		return fmt.Errorf("unsupported protocol %q: must be netflow or ipfix", protocol)
	}
}

// resolveCredentials returns the username and hashed password for the web server.
// It checks CLI flags, then environment variables, then generates a random password.
func resolveCredentials(cliUsername, cliPassword string) (string, string, error) {
	username := cliUsername
	if username == "" {
		username = os.Getenv("FLOWGRE_WEB_USERNAME")
	}
	if username == "" {
		username = "admin"
	}

	password := cliPassword
	if password == "" {
		password = os.Getenv("FLOWGRE_WEB_PASSWORD")
	}

	if password == "" {
		var err error
		password, err = web.GenerateRandomPassword(16)
		if err != nil {
			return "", "", fmt.Errorf("generate random web password: %w", err)
		}
		log.Printf("Generated random web password: %s", password)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("generate web password hash: %w", err)
	}

	return username, string(hashedPassword), nil
}

// effectiveWebIP returns the effective web IP address, defaulting empty string to loopback.
func effectiveWebIP(webIP string) string {
	if webIP == "" {
		return "127.0.0.1"
	}
	return webIP
}

// validateTemplateInterval returns an error if the interval is negative.
func validateTemplateInterval(interval int) error {
	if interval < 0 {
		return fmt.Errorf("template-interval must be 0 (disabled) or a positive value, got %d", interval)
	}
	return nil
}

// validateWebBinding checks that the web server binding is safe.
// Non-loopback addresses require explicit credentials.
func validateWebBinding(webIP, cliUsername, cliPassword string) error {
	effective := effectiveWebIP(webIP)

	ip := net.ParseIP(effective)
	if ip == nil {
		return fmt.Errorf("invalid web-ip address: %s", effective)
	}

	// Loopback addresses are always safe
	if ip.IsLoopback() {
		return nil
	}

	// Non-loopback requires explicit credentials
	hasExplicitCreds := cliUsername != "" || cliPassword != "" ||
		os.Getenv("FLOWGRE_WEB_USERNAME") != "" || os.Getenv("FLOWGRE_WEB_PASSWORD") != ""

	if !hasExplicitCreds {
		return fmt.Errorf("binding web server to non-loopback address %s requires explicit credentials via --web-username/--web-password or FLOWGRE_WEB_USERNAME/FLOWGRE_WEB_PASSWORD environment variables", effective)
	}

	return nil
}

// Execute runs the barrage mode with parsed flags.
func (c *BarrageCommand) Execute() error {
	var cfg *models.Config

	// Load configuration from file or CLI flags
	if *c.configFile != "" {
		fmt.Println("Reading config file... ignoring any other given arguments")
		if err := config.InitViper(*c.configFile); err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		var err error
		cfg, err = config.LoadBarrageConfig()
		if err != nil {
			return fmt.Errorf("error loading barrage config: %w", err)
		}
	} else {
		cfg = &models.Config{
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
			WebUsername:      *c.webUsername,
			WebPassword:      *c.webPassword,
		}
	}

	// Validate protocol
	if err := validateProtocol(cfg.Protocol); err != nil {
		return err
	}

	// Validate template interval
	if err := validateTemplateInterval(cfg.TemplateInterval); err != nil {
		return err
	}

	// Validate web binding safety
	if cfg.Web {
		if err := validateWebBinding(cfg.WebIP, cfg.WebUsername, cfg.WebPassword); err != nil {
			return err
		}
	}

	// Resolve credentials BEFORE starting workers so errors fail fast
	var webUsername, webHashedPassword string
	if cfg.Web {
		var err error
		webUsername, webHashedPassword, err = resolveCredentials(cfg.WebUsername, cfg.WebPassword)
		if err != nil {
			return fmt.Errorf("resolve web credentials: %w", err)
		}
	}

	// Select generator based on protocol
	var gen barrage.FlowGenerator
	if cfg.Protocol == "ipfix" {
		gen = barrage.IPFIX()
	} else {
		nfProfile := resolveProfile(*c.profile)
		gen = barrage.NetFlow(nfProfile)
	}

	// Setup lifecycle and signal handling
	mgr := lifecycle.New()
	cleanupDone := mgr.SetupSignalHandler()

	go func() {
		<-cleanupDone
		log.Printf("Received signal, shutting down...\n")
		mgr.Cancel()
	}()

	// Start barrage workers
	opts := barrage.StartCtx(mgr.Context(), cfg, gen)

	// Start web server if needed
	if cfg.Web {
		opts.Wg.Add(1)
		effectiveIP := effectiveWebIP(cfg.WebIP)
		go web.RunWebServer(effectiveIP, cfg.WebPort, opts.Wg, mgr.Context(), opts.Stats, webUsername, webHashedPassword)
	}

	opts.Wg.Wait()
	opts.StopFn()
	mgr.Wait()
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
