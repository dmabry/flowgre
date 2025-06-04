package cmd

import (
	"fmt"
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/flow/ipfix"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sync"
	"time"
	"strings"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type IPFIXConfig struct {
	Enabled            bool
	Port               int
	TemplateTimeout    time.Duration
	MaxFlowsPerPacket  int
	AllowedEnterprises []uint32
}

// startIPFIXCollector starts the IPFIX UDP collector
func startIPFIXCollector(config IPFIXConfig) {
	// Set up UDP address
	addr := fmt.Sprintf(":%d", config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Printf("Error resolving UDP address: %v\n", err)
		return
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Printf("Error creating UDP listener: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("IPFIX collector started on port %d\n", config.Port)

	// Buffer for incoming packets
	buf := make([]byte, 65535) // Max UDP packet size

	for {
		// Read packet
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error reading UDP packet: %v\n", err)
			continue
		}

		// Increment packet counter
		IPFIXPacketsReceived.Inc()

		// Process packet in a goroutine to avoid blocking
		go func(data []byte, addr *net.UDPAddr) {
			// Parse IPFIX header
			var header ipfix.Header
			if err := header.FromBytes(data[:n]); err != nil {
				fmt.Printf("Error parsing IPFIX header: %v\n", err)
				return
			}

			// Validate enterprise IDs in template records
			if !header.IsValid() {
				fmt.Printf("Invalid IPFIX header from %v\n", addr)
				return
			}

			// Process template records
			// This is a simplified example - actual implementation would parse the entire packet
			// and check all enterprise IDs in template records
			// For this example, we'll assume the template record starts after the header
			if n > 16 {
				// Skip header (16 bytes)
				templateData := data[16:n]
				
				// Parse template records
				// This is a simplified example - actual implementation would parse all records
				var template ipfix.TemplateRecord
				if err := template.FromBytes(templateData); err != nil {
					fmt.Printf("Error parsing IPFIX template: %v\n", err)
					return
				}

				// Check if enterprise ID is allowed
				allowed := false
				for _, allowedID := range config.AllowedEnterprises {
					for _, field := range template.Fields {
						if field.EnterpriseID == allowedID {
							allowed = true
							break
						}
					}
					if allowed {
						break
					}
				}

				if !allowed && len(config.AllowedEnterprises) > 0 {
					IPFIXInvalidEnterprise.Inc()
					fmt.Printf("Packet from %v contains disallowed enterprise ID\n", addr)
					return
				}
			}

			// Process data flow sets (simplified)
			// In a real implementation, this would parse and process the actual flow data
			// For this example, we'll just acknowledge receipt
			fmt.Printf("Received valid IPFIX packet from %v, length: %d\n", addr, n)
		}(buf[:n], remoteAddr)
	}
}

var (
	barrageServer   string
	barrageDstPort  int
	barrageSrcRange string
	barrageDstRange string
	ipfixEnabled    bool
	ipfixPort       int
	ipfixTemplateTimeout int
	ipfixMaxFlowsPerPacket int
	ipfixAllowedEnterprises string
	IPFIXPacketsReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "flowgre_ipfix_packets_received_total",
			Help: "Total number of IPFIX packets received",
		})
	IPFIXInvalidEnterprise = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "flowgre_ipfix_invalid_enterprise_id_total",
			Help: "Number of packets with invalid enterprise IDs",
		})
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
				
// Parse IPFIX configuration
	var ipfixConfig IPFIXConfig
	if ipfixEnabled {
		// Parse allowed enterprises
		allowedEnterprises := strings.Split(ipfixAllowedEnterprises, ",")
		enterpriseIDs := make([]uint32, 0, len(allowedEnterprises))
		
		for _, eidStr := range allowedEnterprises {
			eidStr = strings.TrimSpace(eidStr)
			if eidStr == "" {
				continue
			}
			
			eid, err := strconv.ParseUint(eidStr, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid enterprise ID: %v", err)
			}
			enterpriseIDs = append(enterpriseIDs, uint32(eid))
		}
		
		ipfixConfig = IPFIXConfig{
			Enabled:            ipfixEnabled,
			Port:               ipfixPort,
			TemplateTimeout:    time.Duration(ipfixTemplateTimeout) * time.Minute,
			MaxFlowsPerPacket:  ipfixMaxFlowsPerPacket,
			AllowedEnterprises: enterpriseIDs,
		}
		
		// Start IPFIX collector
		go startIPFIXCollector(ipfixConfig)
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
	// IPFIX configuration flags
	barrageCmd.Flags().BoolVar(&ipfixEnabled, "ipfix.enabled", false, "Enable IPFIX support")
	barrageCmd.Flags().IntVar(&ipfixPort, "ipfix.port", 4739, "IPFIX listening port")
	barrageCmd.Flags().IntVar(&ipfixTemplateTimeout, "ipfix.template-timeout", 30, "Template timeout in minutes")
	barrageCmd.Flags().IntVar(&ipfixMaxFlowsPerPacket, "ipfix.max-flows-per-packet", 1000, "Maximum number of flows per packet")
	barrageCmd.Flags().StringVar(&ipfixAllowedEnterprises, "ipfix.allowed-enterprises", "0", "Comma-separated list of allowed enterprise IDs")

	rootCmd.AddCommand(barrageCmd)
}