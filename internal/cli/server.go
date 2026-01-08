package cli

import (
	"github.com/gwuah/piko/internal/server"
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
	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	srv := server.New(serverPort, ctx.DB)
	return srv.Start()
}
