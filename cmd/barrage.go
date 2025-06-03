package cmd

import (
	"fmt"
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sync"
)

var (
	barrageServer   string
	barrageDstPort  int
	barrageSrcRange string
	barrageDstRange string
	barrageWorkers  int
	barrageDelay    int
	barrageConfig   string
	barrageWeb      bool
	barrageWebIP    string
	barrageWebPort  int
)

func init() {
	barrageCmd := &cobra.Command{
		Use:   "barrage",
		Short: "Send a continuous barrage of flows to a collector for testing",
		Long: `Barrage is used to send a continuous barrage of flows in different sequences to a collector for testing.
Example: flowgre barrage -server 127.0.0.1 -port 9995 -workers 4 -delay 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse config if given and ignore all other arguments
			if barrageConfig != "" {
				viper.SetConfigFile(barrageConfig)
				if err := viper.ReadInConfig(); err != nil {
					return fmt.Errorf("error reading config file: %v", err)
				}
				
				// Parse the config structure
				if !viper.InConfig("targets") {
					return fmt.Errorf("error couldn't find targets section in given yaml config file")
				}
				
				targets := viper.AllSettings()
				if len(targets) > 1 {
					return fmt.Errorf("found more than 1 target in config file, only 1 is allowed")
				}
				
				for _, value := range targets {
					if v, ok := value.(map[string]interface{}); ok {
						for targetName, targetValues := range v {
							if t, ok := targetValues.(map[string]interface{}); ok {
								targetIP := t["ip"].(string)
								targetPort := t["port"].(int)
								targetWorkers := t["workers"].(int)
								targetDelay := t["delay"].(int)
								
								bConfig := models.Config{
									Server:    targetIP,
									DstPort:   targetPort,
									Workers:   targetWorkers,
									Delay:     targetDelay,
									WaitGroup: sync.WaitGroup{},
								}
								
								fmt.Printf("target: %s ip: %s port: %d workers: %d delay: %d\n",
									targetName, targetIP, targetPort, targetWorkers, targetDelay)
								barrage.Run(&bConfig)
							}
						}
					}
				}
			} else {
				// Run with command line args
				bConfig := models.Config{
					Server:   barrageServer,
					DstPort:  barrageDstPort,
					SrcRange: barrageSrcRange,
					DstRange: barrageDstRange,
					Delay:    barrageDelay,
					Workers:  barrageWorkers,
					Web:      barrageWeb,
					WebIP:    barrageWebIP,
					WebPort:  barrageWebPort,
				}
				barrage.Run(&bConfig)
			}
			return nil
		},
	}

	barrageCmd.Flags().StringVarP(&barrageServer, "server", "s", "127.0.0.1", "Servername or IP address of the flow collector")
	barrageCmd.Flags().IntVarP(&barrageDstPort, "port", "p", 9995, "Destination port used by the flow collector")
	barrageCmd.Flags().StringVar(&barrageSrcRange, "src-range", "10.0.0.0/8", "CIDR range to use for generating source IPs for flows")
	barrageCmd.Flags().StringVar(&barrageDstRange, "dst-range", "10.0.0.0/8", "CIDR range to use for generating destination IPs for flows")
	barrageCmd.Flags().IntVar(&barrageWorkers, "workers", 4, "Number of workers to create. Unique sources per worker")
	barrageCmd.Flags().IntVar(&barrageDelay, "delay", 100, "Number of milliseconds between packets sent")
	barrageCmd.Flags().StringVar(&barrageConfig, "config", "", "Config file to use. Supersedes all given args")
	barrageCmd.Flags().IntVar(&barrageWebPort, "web-port", 8080, "Port to bind the web server on")
	barrageCmd.Flags().StringVar(&barrageWebIP, "web-ip", "0.0.0.0", "IP address the web server will listen on")
	barrageCmd.Flags().BoolVar(&barrageWeb, "web", false, "Whether to use the web server or not")

	rootCmd.AddCommand(barrageCmd)
}