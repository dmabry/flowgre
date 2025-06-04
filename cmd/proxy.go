package cmd

import (
	"github.com/dmabry/flowgre/proxy"
	"github.com/spf13/cobra"
	"strings"
	"strconv"
	"time"
)

var (
	proxyIP               string
	proxyPort             int
	proxyVerbose          bool
	proxyTargets          []string
	ipfixEnabled          bool
	ipfixPort             int
	ipfixTemplateTimeout  int
	ipfixMaxFlowsPerPacket int
	ipfixAllowedEnterprises string
)

func init() {
	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Accept flows and relay them to multiple targets",
		Long: `Proxy is used to accept flows and relay them to multiple targets.
Example: flowgre proxy -ip 127.0.0.1 -port 9995 -target 192.168.1.1:9995 -target 192.168.1.2:9995`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse IPFIX configuration
			var ipfixConfig ipfix.IPFIXConfig
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
				
				ipfixConfig = ipfix.IPFIXConfig{
					Enabled:            ipfixEnabled,
					Port:               ipfixPort,
					TemplateTimeout:    time.Duration(ipfixTemplateTimeout) * time.Minute,
					MaxFlowsPerPacket:  ipfixMaxFlowsPerPacket,
					AllowedEnterprises: enterpriseIDs,
				}
				
				// Start IPFIX collector
				go ipfix.StartIPFIXCollector(ipfixConfig)
			}
			
			proxy.Run(proxyIP, proxyPort, proxyVerbose, proxyTargets)
			return nil
		},
	}

	proxyCmd.Flags().StringVarP(&proxyIP, "ip", "i", "127.0.0.1", "IP address proxy should listen on")
	proxyCmd.Flags().IntVarP(&proxyPort, "port", "p", 9995, "Proxy listen UDP port")
	proxyCmd.Flags().StringArrayVar(&proxyTargets, "target", []string{}, "Target addresses in IP:PORT format (can be specified multiple times)")
	proxyCmd.Flags().BoolVar(&proxyVerbose, "verbose", false, "Log every flow received")
	// IPFIX configuration flags
	proxyCmd.Flags().BoolVar(&ipfixEnabled, "ipfix.enabled", false, "Enable IPFIX support")
	proxyCmd.Flags().IntVar(&ipfixPort, "ipfix.port", 4739, "IPFIX listening port")
	proxyCmd.Flags().IntVar(&ipfixTemplateTimeout, "ipfix.template-timeout", 30, "Template timeout in minutes")
	proxyCmd.Flags().IntVar(&ipfixMaxFlowsPerPacket, "ipfix.max-flows-per-packet", 1000, "Maximum number of flows per packet")
	proxyCmd.Flags().StringVar(&ipfixAllowedEnterprises, "ipfix.allowed-enterprises", "0", "Comma-separated list of allowed enterprise IDs")

	rootCmd.AddCommand(proxyCmd)
}