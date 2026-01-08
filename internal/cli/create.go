package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/git"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new worktree environment",
	Long:  `Create a new isolated development environment with its own git worktree and Docker containers.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

var createBranch string

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBranch, "branch", "", "Use existing branch instead of creating new")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 1. Validate initialized
	dbPath := filepath.Join(cwd, ".piko", "state.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}

	// 2. Open database
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 3. Get project
	project, err := db.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// 4. Check name not used
	exists, err := db.EnvironmentExists(name)
	if err != nil {
		return fmt.Errorf("failed to check environment: %w", err)
	}
	if exists {
		return fmt.Errorf("environment %q already exists (use 'piko destroy %s' first)", name, name)
	}

	// 5. Load config
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 6. Create worktrees directory
	worktreesDir := filepath.Join(cwd, ".piko", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// 7. Create worktree
	wtOpts := git.WorktreeOptions{
		Name:       name,
		BasePath:   worktreesDir,
		BranchName: createBranch,
	}
	wt, err := git.CreateWorktree(wtOpts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	fmt.Printf("✓ Created worktree at %s (branch: %s)\n", wt.Path, wt.Branch)

	// Helper to clean up on failure
	cleanup := func() {
		git.RemoveWorktree(wt.Path)
	}

	// 8. Parse compose config from worktree
	composeConfig, err := docker.ParseComposeConfig(wt.Path)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	// 9. Insert environment (to get ID for port allocation)
	dockerProject := fmt.Sprintf("piko-%s-%s", project.Name, name)
	environment := &state.Environment{
		ProjectID:     project.ID,
		Name:          name,
		Branch:        wt.Branch,
		Path:          wt.Path,
		DockerProject: dockerProject,
	}
	envID, err := db.InsertEnvironment(environment)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to save environment: %w", err)
	}
	environment.ID = envID

	// Helper to clean up environment record
	cleanupWithDB := func() {
		db.DeleteEnvironment(name)
		cleanup()
	}

	// 10. Allocate ports
	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(envID, servicePorts)

	// 11. Generate override file
	override := docker.GenerateOverride(project.Name, name, allocations)
	overridePath := filepath.Join(wt.Path, "docker-compose.piko.yml")
	if err := docker.WriteOverrideFile(overridePath, override); err != nil {
		cleanupWithDB()
		return fmt.Errorf("failed to write override file: %w", err)
	}
	fmt.Println("✓ Generated docker-compose.piko.yml")

	// 12. Start containers
	composeCmd := exec.Command("docker", "compose",
		"-p", dockerProject,
		"-f", "docker-compose.yml",
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = wt.Path
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		cleanupWithDB()
		return fmt.Errorf("failed to start containers: %w", err)
	}
	fmt.Printf("✓ Started containers (%s)\n", dockerProject)

	// Helper to stop containers on failure
	cleanupWithContainers := func() {
		stopCmd := exec.Command("docker", "compose", "-p", dockerProject, "down")
		stopCmd.Dir = wt.Path
		stopCmd.Run()
		cleanupWithDB()
	}

	// 13. Run setup script
	if cfg.Scripts.Setup != "" {
		pikoEnv := env.Build(project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		fmt.Println("Running setup script...")
		if err := runner.RunSetup(cfg.Scripts.Setup); err != nil {
			cleanupWithContainers()
			return fmt.Errorf("setup script failed: %w", err)
		}
		fmt.Println("✓ Ran setup script")
	}

	fmt.Println("✓ Environment ready")
	return nil
}
