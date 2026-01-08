package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	dbPath := filepath.Join(cwd, ".piko", "state.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}

	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	srv := server.New(serverPort, db, cwd)
	return srv.Start()
}
