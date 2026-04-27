package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

// ResolveUse resolves `use` fields in project declarations.
// For each project where Use != "", it parses the use file and merges
// fn/seq/par/var declarations into cfg, then runs full validation.
func ResolveUse(cfg *Config) error {
	for _, proj := range cfg.Projects {
		if proj.Use == "" {
			continue
		}
		// Skip projects with sync=ignore — they may not be cloned locally,
		// so their use file won't exist on disk
		if proj.Sync == "ignore" {
			continue
		}
		usePath := filepath.Join(cfg.Sanctuary, proj.Dir, proj.Use)

		// Check if use file exists
		if _, err := os.Stat(usePath); os.IsNotExist(err) {
			return fmt.Errorf("project %q: use file not found: %s (run 'sro sync' first)", proj.Name, usePath)
		}

		// Parse the use file with a fresh visited map
		visited := map[string]bool{}
		programs, err := parseRecursive(usePath, visited)
		if err != nil {
			return fmt.Errorf("project %q: failed to parse use file %s: %v", proj.Name, usePath, err)
		}

		// Validate use file doesn't contain sanctuary or pr blocks
		for _, prog := range programs {
			for _, stmt := range prog.Statements {
				switch stmt.(type) {
				case *ast.SanctuaryDecl:
					return fmt.Errorf("project %q: use file %s cannot declare sanctuary", proj.Name, usePath)
				case *ast.ProjectDecl:
					return fmt.Errorf("project %q: use file %s cannot declare projects", proj.Name, usePath)
				}
			}
		}

		// Merge use file declarations into config
		useCfg, err := merge(programs)
		if err != nil {
			return fmt.Errorf("project %q: failed to merge use file %s: %v", proj.Name, usePath, err)
		}

		// Merge fn/seq/par/vars from use file
		for name, fn := range useCfg.Functions {
			if _, exists := cfg.Functions[name]; exists {
				return fmt.Errorf("project %q: duplicate function %q from use file %s", proj.Name, name, usePath)
			}
			cfg.Functions[name] = fn
		}
		for name, seq := range useCfg.Seqs {
			if _, exists := cfg.Seqs[name]; exists {
				return fmt.Errorf("project %q: duplicate seq %q from use file %s", proj.Name, name, usePath)
			}
			cfg.Seqs[name] = seq
		}
		for name, par := range useCfg.Pars {
			if _, exists := cfg.Pars[name]; exists {
				return fmt.Errorf("project %q: duplicate par %q from use file %s", proj.Name, name, usePath)
			}
			cfg.Pars[name] = par
		}
		for name, val := range useCfg.Vars {
			if _, exists := cfg.Vars[name]; exists {
				return fmt.Errorf("project %q: duplicate var %q from use file %s", proj.Name, name, usePath)
			}
			cfg.Vars[name] = val
		}
	}

	// Run full validation after all use files are merged
	if err := validateFull(cfg); err != nil {
		return err
	}

	return nil
}
