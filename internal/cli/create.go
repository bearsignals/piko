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
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new worktree environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

var (
	createBranch   string
	createNoAttach bool
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBranch, "branch", "", "Use existing branch instead of creating new")
	createCmd.Flags().BoolVar(&createNoAttach, "no-attach", false, "Don't attach to tmux session after creation")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
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

	project, err := db.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	exists, err := db.EnvironmentExists(name)
	if err != nil {
		return fmt.Errorf("failed to check environment: %w", err)
	}
	if exists {
		return fmt.Errorf("environment %q already exists (use 'piko destroy %s' first)", name, name)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	worktreesDir := filepath.Join(cwd, ".piko", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

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

	cleanup := func() {
		git.RemoveWorktree(wt.Path)
	}

	composeDir := wt.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(wt.Path, project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	dockerProject := fmt.Sprintf("piko-%s-%s", project.Name, name)
	sessionName := tmux.SessionName(project.Name, name)

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

	cleanupWithDB := func() {
		db.DeleteEnvironment(name)
		cleanup()
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(envID, servicePorts)

	override := docker.GenerateOverride(project.Name, name, allocations)
	overridePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	if err := docker.WriteOverrideFile(overridePath, override); err != nil {
		cleanupWithDB()
		return fmt.Errorf("failed to write override file: %w", err)
	}
	fmt.Println("✓ Generated docker-compose.piko.yml")

	if cfg.Scripts.Prepare != "" {
		pikoEnv := env.Build(project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		fmt.Println("Running prepare script...")
		if err := runner.RunPrepare(cfg.Scripts.Prepare); err != nil {
			cleanupWithDB()
			return fmt.Errorf("prepare script failed: %w", err)
		}
		fmt.Println("✓ Ran prepare script")
	}

	composeCmd := exec.Command("docker", "compose",
		"-p", dockerProject,
		"-f", "docker-compose.yml",
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		cleanupWithDB()
		return fmt.Errorf("failed to start containers: %w", err)
	}
	fmt.Printf("✓ Started containers (%s)\n", dockerProject)

	cleanupWithContainers := func() {
		stopCmd := exec.Command("docker", "compose", "-p", dockerProject, "down")
		stopCmd.Dir = composeDir
		stopCmd.Run()
		cleanupWithDB()
	}

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

	services := composeConfig.GetServiceNames()
	tmuxCfg := tmux.SessionConfig{
		SessionName:   sessionName,
		WorkDir:       wt.Path,
		DockerProject: dockerProject,
		Services:      services,
		Shells:        cfg.Shells,
	}

	if err := tmux.CreateFullSession(tmuxCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create tmux session: %v\n", err)
	} else {
		fmt.Printf("✓ Created tmux session %s\n", sessionName)
	}

	fmt.Println("✓ Environment ready")

	if !createNoAttach && tmux.SessionExists(sessionName) {
		fmt.Println("→ Attaching...")
		return tmux.Attach(sessionName)
	}

	return nil
}
