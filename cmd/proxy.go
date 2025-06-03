package cmd

import (
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/proxy"
	"github.com/spf13/cobra"
)

var (
	proxyIP       string
	proxyPort     int
	proxyVerbose  bool
	proxyTargets  []string
)

func init() {
	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Accept flows and relay them to multiple targets",
		Long: `Proxy is used to accept flows and relay them to multiple targets.
Example: flowgre proxy -ip 127.0.0.1 -port 9995 -target 192.168.1.1:9995 -target 192.168.1.2:9995`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return proxy.Run(proxyIP, proxyPort, proxyVerbose, proxyTargets)
		},
	}

	proxyCmd.Flags().StringVarP(&proxyIP, "ip", "i", "127.0.0.1", "IP address proxy should listen on")
	proxyCmd.Flags().IntVarP(&proxyPort, "port", "p", 9995, "Proxy listen UDP port")
	proxyCmd.Flags().StringArrayVar(&proxyTargets, "target", []string{}, "Target addresses in IP:PORT format (can be specified multiple times)")
	proxyCmd.Flags().BoolVar(&proxyVerbose, "verbose", false, "Log every flow received")

	rootCmd.AddCommand(proxyCmd)
}