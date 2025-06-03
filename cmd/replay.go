package cmd

import (
	"github.com/dmabry/flowgre/replay"
	"github.com/spf13/cobra"
)

var (
	replayServer   string
	replayPort     int
	replayDelay    int
	replayDB       string
	replayLoop     bool
	replayWorkers  int
	replayUpdateTS bool
	replayVerbose  bool
)

func init() {
	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay recorded flows to a target server",
		Long: `Replay is used to send recorded flows to a target server.
Example: flowgre replay -server 127.0.0.1 -port 9995 -delay 100 -db recorded_flows`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return replay.Run(replayServer, replayPort, replayDelay, replayDB, replayLoop, replayWorkers, replayUpdateTS, replayVerbose)
		},
	}

	replayCmd.Flags().StringVarP(&replayServer, "server", "s", "127.0.0.1", "Target server to replay flows at")
	replayCmd.Flags().IntVarP(&replayPort, "port", "p", 9995, "Target server UDP port")
	replayCmd.Flags().IntVar(&replayDelay, "delay", 100, "Number of milliseconds between packets sent")
	replayCmd.Flags().StringVar(&replayDB, "db", "recorded_flows", "Directory to read recorded flows from")
	replayCmd.Flags().BoolVar(&replayLoop, "loop", false, "Loops the replays forever")
	replayCmd.Flags().IntVar(&replayWorkers, "workers", 1, "Number of workers to spawn for replay")
	replayCmd.Flags().BoolVar(&replayUpdateTS, "updatets", false, "Update to current timestamp on replayed flows")
	replayCmd.Flags().BoolVar(&replayVerbose, "verbose", false, "Log every packet received")

	rootCmd.AddCommand(replayCmd)
}