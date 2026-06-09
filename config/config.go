package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ProjectDirectories []string `toml:"projectDirectories"`
}

func Load() (Config, error) {
	configPath, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("Error reading config %s: %w", configPath, err)
	}

	var parsedConfig Config
	if err := toml.Unmarshal(data, &parsedConfig); err != nil {
		return Config{}, fmt.Errorf("Error parsing config %s: %w", configPath, err)
	}

	return parsedConfig, nil
}

func configPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "tmux-sessions", "config.toml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tmux-sessions", "config.toml"), nil
}
