// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Flowgre is a tool used to generate netflow traffic for testing Netflow collectors.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/dmabry/flowgre/flow/netflow"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

// TODO: Better error handling
func main() {

	// Single SubCommand setup
	singleCmd := flag.NewFlagSet("single", flag.ExitOnError)
	singleCmd.Usage = func() {
		printHelpHeader()
		fmt.Println("Single is used to send a given number of flows in sequence to a collector for testing.")
		fmt.Println("Right now, Source and Destination IPs are randomly generated in the 10.0.0.0/8 range.")
		fmt.Println()
		fmt.Fprintf(singleCmd.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println()
		singleCmd.PrintDefaults()
	}
	singleServer := singleCmd.String("server", "localhost", "servername or ip address of flow collector.")
	singleDstPort := singleCmd.Int("port", 9995, "destination port used by the flow collector.")
	singleSrcPort := singleCmd.Int("srcport", 0, "source port used by the client. If 0 a Random port between 10000-15000")
	singleCount := singleCmd.Int("count", 1, "count of flow to send in sequence.")
	singleHexDump := singleCmd.Bool("hexdump", false, "If true, do a hexdump of the packet")

	// Barrage SubCommand setup
	barrageCmd := flag.NewFlagSet("barrage", flag.ExitOnError)
	barrageCmd.Usage = func() {
		printHelpHeader()
		fmt.Println("COMING SOON!")
		fmt.Println("Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.")
		fmt.Println()
		fmt.Fprintf(barrageCmd.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println()
		barrageCmd.PrintDefaults()
	}

	// Start parsing command line args
	if len(os.Args) < 2 {
		printHelpHeader()
		fmt.Println("Expected 'single' or 'barrage' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {

	// Setup and run Single
	case "single":
		singleCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'single'")
		fmt.Println("  server:", *singleServer)
		fmt.Println("  port:", *singleDstPort)
		fmt.Println("  srcPort:", *singleSrcPort)
		fmt.Println("  count:", *singleCount)
		fmt.Println("  hexdump:", *singleHexDump)
		fmt.Println()

		singleRun(*singleServer, *singleDstPort, *singleSrcPort, *singleCount, *singleHexDump)
		os.Exit(0)

	// Setup and run Barrage
	case "barrage":
		barrageCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'barrage'")
		fmt.Println("COMING SOON!!!")
		os.Exit(404)

	// Shouldn't get here, but if we do it is an error for sure.
	default:
		printHelpHeader()
		fmt.Println("expected 'single' or 'barrage' subcommands")
		os.Exit(2)
	}
	os.Exit(0)

}

// singleRun Creates the given number of Netflow packets, including the required Template, for a Single run.  Creates the packets
// and puts the on the wire to the targeted host.
func singleRun(collectorIP string, destPort int, srcPort int, count int, hexDump bool) {
	printHelpHeader()
	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	if srcPort == 0 {
		rand.Seed(time.Now().UnixNano())
		//Pick random source port between 10000 and 15000
		srcPort = rand.Intn(15000-10000) + 10000
	} // else use the given srcPort number

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(collectorIP)

	// Generate and send Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow()
	tBuf := tFlow.ToBytes()
	fmt.Printf("\nSending Template Flow\n\n")
	fmt.Println(netflow.GetNetFlowSizes(tFlow))
	if hexDump {
		fmt.Printf("%s", hex.Dump(tBuf.Bytes()))
	}
	sendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, tBuf)

	// Generate and send Data Flow(s)
	fmt.Printf("\nSending Data Flows\n\n")
	for i := 1; i <= count; i++ {
		flow := netflow.GenerateDataNetflow(10)
		buf := flow.ToBytes()
		fmt.Println(netflow.GetNetFlowSizes(flow))
		if hexDump {
			fmt.Printf("%s", hex.Dump(buf.Bytes()))
		}
		sendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, buf)
	}
}

// sendPacket Takes a given byte stream and puts on the wire towards the given host
func sendPacket(conn *net.UDPConn, addr *net.UDPAddr, data bytes.Buffer) {
	n, err := conn.WriteTo(data.Bytes(), addr)
	if err != nil {
		log.Fatal("Write:", err)
	}
	fmt.Println("Sent", n, "bytes", conn.LocalAddr(), "->", addr)
}

// printHelpHeader Generates the help header
func printHelpHeader() {
	fmt.Printf("\n   ___ _                             \n  / __\\ | _____      ____ _ _ __ ___ \n / _\\ | |/ _" +
		" \\ \\ /\\ / / _` | '__/ _ \\\n/ /   | | (_) \\ V  V / (_| | | |  __/\n\\/    |_|\\___/ \\_/\\_/ \\__, |_|  \\" +
		"___|\n                      |___/          \n")
	fmt.Println("Slinging packets since 2022!")
	fmt.Println("Used for Netflow Collector Stress testing and other fun activities.")
	fmt.Println()
}
