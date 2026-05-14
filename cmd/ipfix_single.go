// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

const (
	ipfixSourcePortMin = 10000
	ipfixSourcePortMax = 15000
	ipfixSourceIDMin   = 100
	ipfixSourceIDMax   = 10000
)

// IPFIXCommand holds flags and state for the ipfix subcommand.
type IPFIXCommand struct {
	server   *string
	port     *int
	srcPort  *int
	count    *int
	hexDump  *bool
	srcRange *string
	dstRange *string
}

// ParseFlags parses command-line flags for the ipfix mode.
func (c *IPFIXCommand) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("ipfix", flag.ExitOnError)
	c.server = fs.String("server", "127.0.0.1", "servername or ip address of flow collector (IPv4 or IPv6)")
	c.port = fs.Int("port", 9995, "destination port used by the flow collector.")
	c.srcPort = fs.Int("src-port", 0, "source port used by the client. If 0 a Random port between 10000-15000")
	c.count = fs.Int("count", 1, "count of flows to send in sequence.")
	c.hexDump = fs.Bool("hexdump", false, "If true, do a hexdump of the packet")
c.srcRange = fs.String("src-range", "10.0.0.0/8", "CIDR range for source IPs (IPv4 or IPv6)")
c.dstRange = fs.String("dst-range", "10.0.0.0/8", "CIDR range for destination IPs (IPv4 or IPv6)")
	return fs.Parse(args)
}

// Execute runs the ipfix mode with parsed flags.
func (c *IPFIXCommand) Execute() {
	if *c.srcPort == 0 {
		*c.srcPort = utils.RandomNum(ipfixSourcePortMin, ipfixSourcePortMax)
	}
	sourceID := utils.RandomNum(ipfixSourceIDMin, ipfixSourceIDMax)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: *c.srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	defer conn.Close()

	destIP := net.ParseIP(*c.server)
	if destIP == nil {
		log.Fatalf("Failed to parse destination IP %s", *c.server)
	}

	session := netflow.NewSession()

	// Generate and send Template Flow
	tFlow := ipfix.GenerateTemplateIPFIX(sourceID, session)
	tBuf := tFlow.ToBytes()
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: *c.port}, tBuf.Bytes(), *c.hexDump)
	if err != nil {
		log.Fatalf("Issue sending IPFIX template: %v", err)
	}

	// Generate and send Data Flows
	for i := 1; i <= *c.count; i++ {
		flow := ipfix.GenerateDataIPFIX(10, sourceID, *c.srcRange, *c.dstRange, 0, session)
		buf := flow.ToBytes()
		_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: *c.port}, buf.Bytes(), *c.hexDump)
		if err != nil {
			log.Fatalf("Issue sending IPFIX data: %v", err)
		}
	}
}

// RunIPFIX is the entry point for the ipfix subcommand.
func RunIPFIX(args []string) {
	c := &IPFIXCommand{}
	if err := c.ParseFlags(args); err != nil {
		os.Exit(1)
	}
	c.Execute()
}
