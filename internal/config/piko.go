package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .piko.yml configuration file.
type Config struct {
	Scripts Scripts           `yaml:"scripts"`
	Shared  []string          `yaml:"shared"`
	Shells  map[string]string `yaml:"shells"`
	Ignore  []string          `yaml:"ignore"`
}

type Scripts struct {
	Prepare string `yaml:"prepare"`
	Setup   string `yaml:"setup"`
	Run     string `yaml:"run"`
	Destroy string `yaml:"destroy"`
}

// Load loads the .piko.yml configuration from the given directory.
// Returns an empty config if the file doesn't exist (scripts are optional).
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, ".piko.yml")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil // Empty config if file doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read .piko.yml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid .piko.yml: %w", err)
	}

	return &cfg, nil
}
