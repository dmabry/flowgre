// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Flowgre is a tool used to generate netflow traffic for testing Netflow collectors.
package main

import (
	"flag"
	"fmt"
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/single"
	"github.com/spf13/viper"
	"log"
	"os"
	"reflect"
	"strconv"
	"sync"
)

const (
	version = "main" // semantic version
	license = "Apache License, Version 2.0"
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
	singleServer := singleCmd.String("server", "127.0.0.1", "servername or ip address of flow collector.")
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
	barrageServer := barrageCmd.String("server", "127.0.0.1", "servername or ip address of the flow collector")
	barrageDstPort := barrageCmd.Int("port", 9995, "destination port used by the flow collector")
	barrageWorkers := barrageCmd.Int("workers", 4, "number of workers to create. Unique sources per worker")
	barrageDelay := barrageCmd.Int("delay", 100, "number of milliseconds between packets sent")
	barrageConfigFile := barrageCmd.String("config", "", "Config file to use.  Supersedes all given args")
	barrageWebPort := barrageCmd.Int("web-port", 8080, "Port to bind the web server on")
	barrageWebIP := barrageCmd.String("web-ip", "0.0.0.0", "IP address the web server will listen on")
	barrageWeb := barrageCmd.Bool("web", false, "Whether to use the web server or not")

	// Start parsing command line args
	if len(os.Args) < 2 {
		printGenericHelp()
		fmt.Println("expected 'single', 'barrage' or 'version' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {

	// Setup and run Single
	case "single":
		printHelpHeader()
		err := singleCmd.Parse(os.Args[2:])
		if err != nil {
			panic(fmt.Errorf("error parsing args: %v\n", err))
		}

		single.Run(*singleServer, *singleDstPort, *singleSrcPort, *singleCount, *singleHexDump)
		os.Exit(0)

	// Setup and run Barrage
	case "barrage":
		printHelpHeader()
		err := barrageCmd.Parse(os.Args[2:])
		if err != nil {
			panic(fmt.Errorf("error parsing args: %v\n", err))
		}
		// Parse config if given and ignore all other arguments
		if *barrageConfigFile != "" {
			log.Printf("Reading config file... ignoring any other given arguments\n\n")
			viper.SetConfigFile(*barrageConfigFile)
			err := viper.ReadInConfig()
			if err != nil {
				panic(fmt.Errorf("error reading config file: %v\n", err))
			}
			// TODO: At some point it would be interesting to be able to define multiple targets.  For now, only
			// TODO: supporting one.
			// Parse the config structure returned by viper with the expected yaml format below
			// targets:
			//  server1:
			//    ip: 127.0.0.1
			//    port: 9995
			//    workers: 4
			//    delay: 100
			if viper.InConfig("targets") {
				targets := viper.AllSettings()
				// fail if more than 1 target is found for now.  In the future, we'll handle more.
				if len(targets) > 1 {
					panic(fmt.Errorf("found more than 1 target in config file, only 1 is allowed"))
				}
				for _, value := range targets {
					// Should be safe to assume that viper always returns map[string]interface{}, but using switch to be
					// 100% sure the value type returned is as expected.
					switch v := value.(type) {
					case map[string]interface{}:
						// targetName, using the example above, is server1 and targetValues are a map of settings
						for targetName, targetValues := range v {
							// TODO: For Debug purposes... I'll figure out logging and debug flags
							// for key, setting := range targetValues.(map[string]interface{}) {
							//	fmt.Printf("target: %s key: %s setting: %s\n", targetName, key, setting.(string))
							//}
							t := targetValues.(map[string]interface{})
							targetIP := t["ip"].(string)
							targetPort := t["port"].(int)
							targetWorkers := t["workers"].(int)
							targetDelay := t["delay"].(int)
							bConfig := barrage.Config{
								Server:    targetIP,
								DstPort:   targetPort,
								Workers:   targetWorkers,
								Delay:     targetDelay,
								WaitGroup: sync.WaitGroup{},
							}

							fmt.Printf("target: %s ip: %s port: %s workers: %s delay: %s\n",
								targetName, targetIP, strconv.Itoa(targetPort),
								strconv.Itoa(targetWorkers), strconv.Itoa(targetDelay))
							barrage.Run(&bConfig)
						}
					default:
						var r = reflect.TypeOf(v)
						panic(fmt.Errorf("error unexpected type returned by viper: %v\n", r))
					}
				}
			} else {
				panic(fmt.Errorf("error couldn't find targets section in given yaml config file"))
			}
		} else {
			// Run with the args given from cmd line
			bConfig := barrage.Config{
				Server:  *barrageServer,
				DstPort: *barrageDstPort,
				Delay:   *barrageDelay,
				Workers: *barrageWorkers,
				Web:     *barrageWeb,
				WebIP:   *barrageWebIP,
				WebPort: *barrageWebPort,
			}

			barrage.Run(&bConfig)
			os.Exit(0)
		}

	case "version":
		printHelpHeader()
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("License: %s\n", license)
	case "help":
		printGenericHelp()
	default:
		printGenericHelp()
		fmt.Println("expected 'single', 'barrage' or 'version' subcommands")
		os.Exit(2)
	}
}

// printHelpHeader Generates the help header
func printHelpHeader() {
	fmt.Printf("\n   ___ _                             \n  / __\\ | _____      ____ _ _ __ ___ \n / _\\ | |/ _" +
		" \\ \\ /\\ / / _` | '__/ _ \\\n/ /   | | (_) \\ V  V / (_| | | |  __/\n\\/    |_|\\___/ \\_/\\_/ \\__, |_|  \\" +
		"___|\n                      |___/          \n")
	fmt.Println("Slinging packets since 2022!")
	fmt.Println("Used for Netflow Collector Stress testing and other fun activities.")
}

// printGenericHelp prints out the top-level generic help
func printGenericHelp() {
	printHelpHeader()
	fmt.Printf("Version: %s\n", version)
	fmt.Println("expected 'single', 'barrage' or 'version' subcommands")
	fmt.Println("to print more details pass '-help' after the subcommand")
	fmt.Println()
	fmt.Println("Single is used to send a given number of flows in sequence to a collector for testing.")
	fmt.Println("Right now, Source and Destination IPs are randomly generated in the 10.0.0.0/8 range.")
	fmt.Println()
	fmt.Println("Barrage is used to send a continuous barrage of flows in different sequence to a collector for testing.")
	fmt.Println()

}
