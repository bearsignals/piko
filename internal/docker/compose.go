package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

var composeFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

func DetectComposeFile(dir string) (string, error) {
	for _, name := range composeFilenames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no compose file found (tried: %v)", composeFilenames)
}

type ComposeConfig struct {
	project *types.Project
}

func ParseComposeConfig(workDir string) (*ComposeConfig, error) {
	filename, err := DetectComposeFile(workDir)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(workDir, filename))
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	configDetails := types.ConfigDetails{
		WorkingDir:  workDir,
		Environment: types.NewMapping(os.Environ()),
		ConfigFiles: []types.ConfigFile{
			{
				Filename: filename,
				Content:  data,
			},
		},
	}

	project, err := loader.LoadWithContext(context.Background(), configDetails,
		func(o *loader.Options) {
			o.SetProjectName(filepath.Base(workDir), false)
			o.SkipValidation = true
			o.SkipResolveEnvironment = true
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose config: %w", err)
	}

	return &ComposeConfig{project: project}, nil
}

func (c *ComposeConfig) GetServicePorts() map[string][]int {
	result := make(map[string][]int)
	for _, svc := range c.project.Services {
		var ports []int
		for _, p := range svc.Ports {
			if p.Target > 0 {
				ports = append(ports, int(p.Target))
			}
		}
		if len(ports) > 0 {
			result[svc.Name] = ports
		}
	}
	return result
}

func (c *ComposeConfig) GetServiceNames() []string {
	names := make([]string, 0, len(c.project.Services))
	for _, svc := range c.project.Services {
		names = append(names, svc.Name)
	}
	return names
}
