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

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	exists, err := ctx.EnvironmentExists(name)
	if err != nil {
		return fmt.Errorf("failed to check environment: %w", err)
	}
	if exists {
		return fmt.Errorf("environment %q already exists (use 'piko destroy %s' first)", name, name)
	}

	cfg, err := config.Load(ctx.Project.RootPath)
	if err != nil {
		cfg = &config.Config{}
	}

	worktreesDir := ctx.Project.WorktreesDir()
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

	dataDir := filepath.Join(ctx.Project.RootPath, ".piko", "data", name)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		git.RemoveWorktree(wt.Path)
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	fmt.Printf("✓ Created data directory at %s\n", dataDir)

	cleanup := func() {
		os.RemoveAll(dataDir)
		git.RemoveWorktree(wt.Path)
	}

	composeDir := wt.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(wt.Path, ctx.Project.ComposeDir)
	}

	_, composeErr := docker.DetectComposeFile(composeDir)
	isSimpleMode := composeErr != nil

	dockerProject := ""
	if !isSimpleMode {
		dockerProject = fmt.Sprintf("piko-%s-%s", ctx.Project.Name, name)
	}
	sessionName := tmux.SessionName(ctx.Project.Name, name)

	environment := &state.Environment{
		ProjectID:     ctx.Project.ID,
		Name:          name,
		Branch:        wt.Branch,
		Path:          wt.Path,
		DockerProject: dockerProject,
	}
	envID, err := ctx.DB.InsertEnvironment(environment)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to save environment: %w", err)
	}
	environment.ID = envID

	cleanupWithDB := func() {
		ctx.DeleteEnvironment(name)
		cleanup()
	}

	var allocations []ports.Allocation

	if isSimpleMode {
		fmt.Println("✓ Simple mode (no docker-compose found)")
	} else {
		composeConfig, err := docker.ParseComposeConfig(composeDir)
		if err != nil {
			cleanupWithDB()
			return fmt.Errorf("failed to parse compose config: %w", err)
		}

		servicePorts := composeConfig.GetServicePorts()
		allocations = ports.Allocate(envID, servicePorts)

		composeProject := composeConfig.Project()
		docker.ApplyOverrides(composeProject, ctx.Project.Name, name, allocations)
		pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
		if err := docker.WriteProjectFile(pikoComposePath, composeProject); err != nil {
			cleanupWithDB()
			return fmt.Errorf("failed to write compose file: %w", err)
		}
		fmt.Println("✓ Generated docker-compose.piko.yml")
	}

	if cfg.Scripts.Prepare != "" {
		pikoEnv := env.Build(ctx.Project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		fmt.Println("Running prepare script...")
		if err := runner.RunPrepare(cfg.Scripts.Prepare); err != nil {
			cleanupWithDB()
			return fmt.Errorf("prepare script failed: %w", err)
		}
		fmt.Println("✓ Ran prepare script")
	}

	cleanupWithContainers := cleanupWithDB
	if !isSimpleMode {
		composeCmd := exec.Command("docker", "compose",
			"-p", dockerProject,
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

		cleanupWithContainers = func() {
			stopCmd := exec.Command("docker", "compose", "-p", dockerProject, "down")
			stopCmd.Dir = composeDir
			stopCmd.Run()
			cleanupWithDB()
		}
	}

	if cfg.Scripts.Setup != "" {
		pikoEnv := env.Build(ctx.Project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		fmt.Println("Running setup script...")
		if err := runner.RunSetup(cfg.Scripts.Setup); err != nil {
			cleanupWithContainers()
			return fmt.Errorf("setup script failed: %w", err)
		}
		fmt.Println("✓ Ran setup script")
	}

	var services []string
	if !isSimpleMode {
		composeConfig, _ := docker.ParseComposeConfig(composeDir)
		if composeConfig != nil {
			services = composeConfig.GetServiceNames()
		}
	}

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
