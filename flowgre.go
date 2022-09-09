// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Flowgre is a tool used to generate netflow traffic for testing Netflow collectors.
package main

import (
	"flag"
	"fmt"
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/single"
	"os"
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
		fmt.Println("Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.")
		fmt.Println()
		fmt.Fprintf(barrageCmd.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println()
		barrageCmd.PrintDefaults()
	}
	barrageServer := barrageCmd.String("server", "localhost", "servername or ip address of the flow collector")
	barrageDstPort := barrageCmd.Int("port", 9995, "destination port used by the flow collector")
	barrageWorkers := barrageCmd.Int("workers", 4, "number of workers to create. Unique sources per worker")
	barrageDelay := barrageCmd.Int("delay", 100, "number of milliseconds between packets sent")

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

		printHelpHeader()
		single.Run(*singleServer, *singleDstPort, *singleSrcPort, *singleCount, *singleHexDump)
		os.Exit(0)

	// Setup and run Barrage
	case "barrage":
		barrageCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'barrage'")
		fmt.Println("  server:", *barrageServer)
		fmt.Println("  port:", *barrageDstPort)
		fmt.Println("  workers:", *barrageWorkers)
		fmt.Println("  delay:", *barrageDelay)
		barrage.Run(*barrageServer, *barrageDstPort, *barrageDelay, *barrageWorkers)
		os.Exit(0)

	// Shouldn't get here, but if we do it is an error for sure.
	default:
		printHelpHeader()
		fmt.Println("expected 'single' or 'barrage' subcommands")
		os.Exit(2)
	}
	os.Exit(0)

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
