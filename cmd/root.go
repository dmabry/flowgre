package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "flowgre",
	Short: "Flowgre is a tool used to generate netflow traffic for testing Netflow collectors",
	Long: `Flowgre is a tool used to generate netflow traffic for testing Netflow collectors.
It supports multiple modes including single, barrage, record, replay, and proxy.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}