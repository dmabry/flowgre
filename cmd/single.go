package cmd

import (
	"github.com/dmabry/flowgre/single"
	"github.com/spf13/cobra"
)

var (
	singleServer   string
	singleDstPort  int
	singleSrcPort  int
	singleCount    int
	singleHexDump  bool
	singleSrcRange string
	singleDstRange string
)

func init() {
	singleCmd := &cobra.Command{
		Use:   "single",
		Short: "Send a given number of flows in sequence to a collector for testing",
		Long: `Single is used to send a given number of flows in sequence to a collector for testing.
Example: flowgre single -server 127.0.0.1 -port 9995 -count 1000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return single.Run(singleServer, singleDstPort, singleSrcPort, singleCount, singleSrcRange, singleDstRange, singleHexDump)
		},
	}

	singleCmd.Flags().StringVarP(&singleServer, "server", "s", "127.0.0.1", "Servername or IP address of flow collector")
	singleCmd.Flags().IntVarP(&singleDstPort, "port", "p", 9995, "Destination port used by the flow collector")
	singleCmd.Flags().IntVar(&singleSrcPort, "src-port", 0, "Source port used by the client. If 0, a random port between 10000-15000")
	singleCmd.Flags().IntVarP(&singleCount, "count", "c", 1, "Count of flows to send in sequence")
	singleCmd.Flags().BoolVar(&singleHexDump, "hexdump", false, "If true, do a hexdump of the packet")
	singleCmd.Flags().StringVar(&singleSrcRange, "src-range", "10.0.0.0/8", "CIDR range to use for generating source IPs for flows")
	singleCmd.Flags().StringVar(&singleDstRange, "dst-range", "10.0.0.0/8", "CIDR range to use for generating destination IPs for flows")

	rootCmd.AddCommand(singleCmd)
}