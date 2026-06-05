// Flowgre is a tool used to generate netflow traffic for testing Netflow collectors.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dmabry/flowgre/cmd"
)

var version = "0.6.0" // semantic version — overridden by -ldflags at build time
const license = "Apache License, Version 2.0"

func main() {
	if len(os.Args) < 2 {
		printGenericHelp()
		fmt.Println("expected 'single', 'barrage', 'ipfix', 'record', 'replay', 'proxy' or 'version' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "single":
		cmd.RunSingle(os.Args[2:])
	case "barrage":
		if err := cmd.RunBarrage(os.Args[2:]); err != nil {
			log.Fatalf("barrage error: %v", err)
		}
	case "ipfix":
		cmd.RunIPFIX(os.Args[2:])
	case "record":
		cmd.RunRecord(os.Args[2:])
	case "replay":
		cmd.RunReplay(os.Args[2:])
	case "proxy":
		cmd.RunProxy(os.Args[2:])
	case "version":
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("License: %s\n", license)
	case "help":
		printGenericHelp()
	default:
		printGenericHelp()
		fmt.Println("expected 'single', 'barrage', 'ipfix', 'record', 'replay', 'proxy' or 'version' subcommands")
		os.Exit(2)
	}
}

func printHelpHeader() {
	fmt.Printf(`   ___ _                             
  / __\ | _____      ____ _ _ __ ___ 
 / _\ | |/ _ \ \ /\ / / _' | '__/ _ \
/ /   | | (_) \ V  V / (_| | | |  __/
\/    |_|\___/ \_/\_/ \__, |_|  \___|
                      |___/          
`)
	fmt.Println("Slinging packets since 2022!")
	fmt.Println("Used for NetFlow v9 and IPFIX (RFC 7011) Collector Stress testing and other fun activities.")
}

func printGenericHelp() {
	printHelpHeader()
	fmt.Printf("Version: %s\n", version)
	fmt.Println()
	fmt.Println("to print more details pass '-help' after the subcommand")
	fmt.Println()
	fmt.Println("Single  - Send a given number of flows in sequence to a collector for testing.")
	fmt.Println("Barrage - Send a continuous barrage of flows to a collector for testing.")
	fmt.Println("IPFIX   - Send IPFIX (RFC 7011) flows to a collector for testing.")
	fmt.Println("Record  - Record flows to a file for later replay testing.")
	fmt.Println("Replay  - Send recorded flows to a target server.")
	fmt.Println("Proxy   - Accept flows and relay them to multiple targets.")
}
