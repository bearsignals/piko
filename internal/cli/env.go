package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Environment management commands",
	Long:  `Commands for managing piko environments.`,
}

var varsCmd = &cobra.Command{
	Use:   "vars <name>",
	Short: "Print PIKO_* environment variables",
	Long:  `Print all PIKO_* environment variables for the given environment. Use with eval: eval $(piko env vars <name>)`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVars,
}

var varsJSON bool

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(varsCmd)
	varsCmd.Flags().BoolVar(&varsJSON, "json", false, "Output as JSON")
}

func runVars(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	portResult, err := docker.DiscoverPorts(resolved.ComposeDir, resolved.Environment.DockerProject)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	pikoEnv := env.Build(resolved.Project, resolved.Environment, portResult.Allocations)

	if varsJSON {
		data, err := pikoEnv.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(pikoEnv.ToShellExport())
	}

	return nil
}
