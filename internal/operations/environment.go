package operations

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
)

type CreateEnvironmentOptions struct {
	DB          *state.DB
	Project     *state.Project
	Name        string
	Branch      string
	Logger      Logger
}

type CreateEnvironmentResult struct {
	Environment *state.Environment
	SessionName string
	IsSimple    bool
	DataDir     string
}

func CreateEnvironment(opts CreateEnvironmentOptions) (*CreateEnvironmentResult, error) {
	log := opts.Logger
	if log == nil {
		log = &SilentLogger{}
	}

	exists, err := opts.DB.EnvironmentExists(opts.Project.ID, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check environment: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("environment %q already exists", opts.Name)
	}

	cfg, err := config.Load(opts.Project.RootPath)
	if err != nil {
		cfg = &config.Config{}
	}

	worktreesDir := opts.Project.WorktreesDir()
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	wtOpts := git.WorktreeOptions{
		Name:       opts.Name,
		BasePath:   worktreesDir,
		BranchName: opts.Branch,
	}
	wt, err := git.CreateWorktree(wtOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}
	log.Infof("Created worktree at %s (branch: %s)", wt.Path, wt.Branch)

	dataDir := filepath.Join(opts.Project.RootPath, ".piko", "data", opts.Name)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		git.RemoveWorktree(wt.Path)
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	log.Infof("Created data directory at %s", dataDir)

	cleanup := func() {
		os.RemoveAll(dataDir)
		git.RemoveWorktree(wt.Path)
	}

	composeDir := wt.Path
	if opts.Project.ComposeDir != "" {
		composeDir = filepath.Join(wt.Path, opts.Project.ComposeDir)
	}

	_, composeErr := docker.DetectComposeFile(composeDir)
	isSimpleMode := composeErr != nil

	dockerProject := ""
	if !isSimpleMode {
		dockerProject = fmt.Sprintf("piko-%s-%s", opts.Project.Name, opts.Name)
	}
	sessionName := tmux.SessionName(opts.Project.Name, opts.Name)

	environment := &state.Environment{
		ProjectID:     opts.Project.ID,
		Name:          opts.Name,
		Branch:        wt.Branch,
		Path:          wt.Path,
		DockerProject: dockerProject,
	}
	envID, err := opts.DB.InsertEnvironment(environment)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to save environment: %w", err)
	}
	environment.ID = envID

	cleanupWithDB := func() {
		opts.DB.DeleteEnvironment(opts.Project.ID, opts.Name)
		cleanup()
	}

	var allocations []ports.Allocation

	if isSimpleMode {
		log.Info("Simple mode (no docker-compose found)")
	} else {
		composeConfig, err := docker.ParseComposeConfig(composeDir)
		if err != nil {
			cleanupWithDB()
			return nil, fmt.Errorf("failed to parse compose config: %w", err)
		}

		servicePorts := composeConfig.GetServicePorts()
		allocations = ports.Allocate(envID, servicePorts)

		composeProject := composeConfig.Project()
		docker.ApplyOverrides(composeProject, opts.Project.Name, opts.Name, allocations)
		pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
		if err := docker.WriteProjectFile(pikoComposePath, composeProject); err != nil {
			cleanupWithDB()
			return nil, fmt.Errorf("failed to write compose file: %w", err)
		}
		log.Info("Generated docker-compose.piko.yml")
	}

	if cfg.Scripts.Prepare != "" {
		pikoEnv := env.Build(opts.Project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		log.Info("Running prepare script...")
		if err := runner.RunPrepare(cfg.Scripts.Prepare); err != nil {
			cleanupWithDB()
			return nil, fmt.Errorf("prepare script failed: %w", err)
		}
		log.Info("Ran prepare script")
	}

	cleanupWithContainers := cleanupWithDB
	if !isSimpleMode {
		composeCmd := exec.Command("docker", "compose",
			"-p", dockerProject,
			"-f", "docker-compose.piko.yml",
			"up", "-d")
		composeCmd.Dir = composeDir

		if err := composeCmd.Run(); err != nil {
			cleanupWithDB()
			return nil, fmt.Errorf("failed to start containers: %w", err)
		}
		log.Infof("Started containers (%s)", dockerProject)

		cleanupWithContainers = func() {
			stopCmd := exec.Command("docker", "compose", "-p", dockerProject, "down")
			stopCmd.Dir = composeDir
			stopCmd.Run()
			cleanupWithDB()
		}
	}

	if cfg.Scripts.Setup != "" {
		pikoEnv := env.Build(opts.Project, environment, allocations)
		runner := config.NewScriptRunner(wt.Path, pikoEnv.ToEnvSlice())

		log.Info("Running setup script...")
		if err := runner.RunSetup(cfg.Scripts.Setup); err != nil {
			cleanupWithContainers()
			return nil, fmt.Errorf("setup script failed: %w", err)
		}
		log.Info("Ran setup script")
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
		log.Warnf("failed to create tmux session: %v", err)
	} else {
		log.Infof("Created tmux session %s", sessionName)
	}

	log.Info("Environment ready")

	return &CreateEnvironmentResult{
		Environment: environment,
		SessionName: sessionName,
		IsSimple:    isSimpleMode,
		DataDir:     dataDir,
	}, nil
}

type DestroyEnvironmentOptions struct {
	DB            *state.DB
	Project       *state.Project
	Environment   *state.Environment
	RemoveVolumes bool
	Logger        Logger
}

func DestroyEnvironment(opts DestroyEnvironmentOptions) error {
	log := opts.Logger
	if log == nil {
		log = &SilentLogger{}
	}

	cfg, err := config.Load(opts.Project.RootPath)
	if err != nil {
		cfg = &config.Config{}
	}

	if cfg.Scripts.Destroy != "" {
		pikoEnv := env.Build(opts.Project, opts.Environment, []ports.Allocation{})
		runner := config.NewScriptRunner(opts.Environment.Path, pikoEnv.ToEnvSlice())

		log.Info("Running destroy script...")
		if err := runner.RunDestroy(cfg.Scripts.Destroy); err != nil {
			log.Warnf("destroy script failed: %v", err)
		}
	}

	sessionName := tmux.SessionName(opts.Project.Name, opts.Environment.Name)
	if tmux.SessionExists(sessionName) {
		if err := tmux.KillSession(sessionName); err != nil {
			log.Warnf("failed to kill tmux session: %v", err)
		} else {
			log.Info("Killed tmux session")
		}
	}

	isSimpleMode := opts.Environment.DockerProject == ""

	if !isSimpleMode {
		composeDir := opts.Environment.Path
		if opts.Project.ComposeDir != "" {
			composeDir = filepath.Join(opts.Environment.Path, opts.Project.ComposeDir)
		}

		var composeCmd *exec.Cmd
		if opts.RemoveVolumes {
			composeCmd = exec.Command("docker", "compose", "-p", opts.Environment.DockerProject, "down", "-v")
		} else {
			composeCmd = exec.Command("docker", "compose", "-p", opts.Environment.DockerProject, "down")
		}
		composeCmd.Dir = composeDir

		if err := composeCmd.Run(); err != nil {
			log.Warnf("failed to stop containers: %v", err)
		} else {
			log.Info("Stopped containers")
			if opts.RemoveVolumes {
				log.Info("Removed volumes")
			}
		}
	}

	if err := git.RemoveWorktree(opts.Environment.Path); err != nil {
		log.Warnf("failed to remove worktree: %v", err)
	} else {
		log.Info("Removed worktree")
	}

	dataDir := filepath.Join(opts.Project.RootPath, ".piko", "data", opts.Environment.Name)
	if err := os.RemoveAll(dataDir); err != nil {
		log.Warnf("failed to remove data directory: %v", err)
	} else {
		log.Info("Removed data directory")
	}

	if err := opts.DB.DeleteEnvironment(opts.Project.ID, opts.Environment.Name); err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}
	log.Info("Removed from database")

	return nil
}

type UpEnvironmentOptions struct {
	DB          *state.DB
	Project     *state.Project
	Environment *state.Environment
	Logger      Logger
}

func UpEnvironment(opts UpEnvironmentOptions) error {
	log := opts.Logger
	if log == nil {
		log = &SilentLogger{}
	}

	if opts.Environment.DockerProject == "" {
		log.Info("Simple mode environment - no containers to start")
		log.Info("Use 'piko attach' to access the tmux session")
		return nil
	}

	composeDir := opts.Environment.Path
	if opts.Project.ComposeDir != "" {
		composeDir = filepath.Join(opts.Environment.Path, opts.Project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(opts.Environment.ID, servicePorts)

	composeProject := composeConfig.Project()
	docker.ApplyOverrides(composeProject, opts.Project.Name, opts.Environment.Name, allocations)
	pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	docker.WriteProjectFile(pikoComposePath, composeProject)

	composeCmd := exec.Command("docker", "compose",
		"-p", opts.Environment.DockerProject,
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = composeDir

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	log.Infof("Started containers (%s)", opts.Environment.DockerProject)
	return nil
}

type DownEnvironmentOptions struct {
	DB          *state.DB
	Project     *state.Project
	Environment *state.Environment
	Logger      Logger
}

func DownEnvironment(opts DownEnvironmentOptions) error {
	log := opts.Logger
	if log == nil {
		log = &SilentLogger{}
	}

	if opts.Environment.DockerProject == "" {
		log.Info("Simple mode environment - no containers to stop")
		return nil
	}

	composeDir := opts.Environment.Path
	if opts.Project.ComposeDir != "" {
		composeDir = filepath.Join(opts.Environment.Path, opts.Project.ComposeDir)
	}

	composeCmd := exec.Command("docker", "compose", "-p", opts.Environment.DockerProject, "down")
	composeCmd.Dir = composeDir

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	log.Info("Stopped containers")
	return nil
}
