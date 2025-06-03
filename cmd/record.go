package cmd

import (
	"github.com/dmabry/flowgre/record"
	"github.com/spf13/cobra"
)

var (
	recordIP     string
	recordPort   int
	recordDB     string
	recordVerbose bool
)

func init() {
	recordCmd := &cobra.Command{
		Use:   "record",
		Short: "Record flows to a file for later replay testing",
		Long: `Record is used to record flows to a file for later replay testing.
Example: flowgre record -ip 127.0.0.1 -port 9995 -db recorded_flows`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return record.Run(recordIP, recordPort, recordDB, recordVerbose)
		},
	}

	recordCmd.Flags().StringVarP(&recordIP, "ip", "i", "127.0.0.1", "IP address record should listen on")
	recordCmd.Flags().IntVarP(&recordPort, "port", "p", 9995, "Listen UDP port")
	recordCmd.Flags().StringVar(&recordDB, "db", "recorded_flows", "Directory to place recorded flows for later replay")
	recordCmd.Flags().BoolVar(&recordVerbose, "verbose", false, "Log every packet received")

	rootCmd.AddCommand(recordCmd)
}