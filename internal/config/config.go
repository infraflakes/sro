package config

import (
	"path/filepath"
)

func Load(entryPath string) (*Config, error) {
	absPath, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, err
	}

	visited := map[string]bool{}
	programs, err := parseRecursive(absPath, visited)
	if err != nil {
		return nil, err
	}

	cfg, err := merge(programs)
	if err != nil {
		return nil, err
	}

	if err := validateBase(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
