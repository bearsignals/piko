package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/server"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the piko web server",
	RunE:  runServer,
}

var serverPort int

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVar(&serverPort, "port", 19876, "Port to listen on")
}

func runServer(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	db, err := state.OpenCentral()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	srv := server.New(serverPort, db)
	return srv.Start()
}
