package config

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

// resolveBacktickLit resolves interpolation in a BacktickLit using the provided vars map.
func resolveBacktickLit(lit *ast.BacktickLit, vars map[string]string) (string, error) {
	return ResolveBacktickLitWithPos(lit, vars, 0, 0)
}

// ResolveBacktickLitWithPos resolves interpolation in a BacktickLit using the provided vars map,
// with optional line/col information for error messages.
func ResolveBacktickLitWithPos(lit *ast.BacktickLit, vars map[string]string, line, col int) (string, error) {
	var sb strings.Builder
	for _, part := range lit.Parts {
		if part.IsVar {
			val, ok := vars[part.Value]
			if !ok {
				if line > 0 || col > 0 {
					return "", fmt.Errorf("%d:%d: undefined variable ${%s}", line, col, part.Value)
				}
				return "", fmt.Errorf("undefined variable: ${%s}", part.Value)
			}
			sb.WriteString(val)
		} else {
			sb.WriteString(part.Value)
		}
	}
	return sb.String(), nil
}

func mergeShellDecl(cfg *Config, s *ast.ShellDecl) error {
	if cfg.Shell != "" {
		return fmt.Errorf("duplicate shell declaration")
	}
	cfg.Shell = s.Value
	return nil
}

func mergeSanctuaryDecl(cfg *Config, s *ast.SanctuaryDecl) error {
	if cfg.Sanctuary != "" {
		return fmt.Errorf("duplicate sanctuary declaration")
	}
	switch v := s.Value.(type) {
	case *ast.BacktickLit:
		resolved, err := resolveBacktickLit(v, cfg.Vars)
		if err != nil {
			return err
		}
		cfg.Sanctuary = resolved
	case *ast.VarRef:
		resolved, ok := cfg.Vars[v.Name]
		if !ok {
			return fmt.Errorf("undefined variable: $%s", v.Name)
		}
		cfg.Sanctuary = resolved
	}
	return nil
}

func mergeVarDecl(cfg *Config, s *ast.VarDecl) error {
	name := s.Name
	if _, exists := cfg.Vars[name]; exists {
		return fmt.Errorf("duplicate variable: %s", name)
	}
	switch v := s.Value.(type) {
	case *ast.BacktickLit:
		resolved, err := resolveBacktickLit(v, cfg.Vars)
		if err != nil {
			return err
		}
		if s.VarType == "shell" {
			if cfg.Shell == "" {
				return fmt.Errorf("shell must be declared before using shell variables")
			}
			cmd := exec.Command(cfg.Shell, "-c", resolved)
			out, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("shell execution failed for var %s: %w", name, err)
			}
			cfg.Vars[name] = strings.TrimRight(string(out), "\n")
		} else {
			cfg.Vars[name] = resolved
		}
	case *ast.VarRef:
		resolved, ok := cfg.Vars[v.Name]
		if !ok {
			return fmt.Errorf("undefined variable: $%s", v.Name)
		}
		cfg.Vars[name] = resolved
	}
	return nil
}

func mergeProjectDecl(cfg *Config, s *ast.ProjectDecl) error {
	if _, exists := cfg.Projects[s.Name]; exists {
		return fmt.Errorf("duplicate project: %s", s.Name)
	}
	proj := &Project{Name: s.Name, Sync: "clone"} // default
	for _, f := range s.Fields {
		var resolved string
		var err error
		switch v := f.Value.(type) {
		case *ast.BacktickLit:
			resolved, err = resolveBacktickLit(v, cfg.Vars)
			if err != nil {
				return err
			}
		case *ast.VarRef:
			var ok bool
			resolved, ok = cfg.Vars[v.Name]
			if !ok {
				return fmt.Errorf("undefined variable: $%s", v.Name)
			}
		}
		switch f.Key {
		case "url":
			proj.URL = resolved
		case "dir":
			proj.Dir = resolved
		case "sync":
			proj.Sync = resolved
		case "use":
			proj.Use = resolved
		}
	}
	cfg.Projects[s.Name] = proj
	return nil
}

func mergeFnDecl(cfg *Config, s *ast.FnDecl) error {
	if _, exists := cfg.Functions[s.Name]; exists {
		return fmt.Errorf("duplicate function: %s", s.Name)
	}
	cfg.Functions[s.Name] = s
	return nil
}

func mergeSeqDecl(cfg *Config, s *ast.SeqDecl) error {
	if _, exists := cfg.Seqs[s.Name]; exists {
		return fmt.Errorf("duplicate seq: %s", s.Name)
	}
	cfg.Seqs[s.Name] = s
	return nil
}

func mergeParDecl(cfg *Config, s *ast.ParDecl) error {
	if _, exists := cfg.Pars[s.Name]; exists {
		return fmt.Errorf("duplicate par: %s", s.Name)
	}
	cfg.Pars[s.Name] = s
	return nil
}

func merge(programs []*ast.Program) (*Config, error) {
	cfg := &Config{
		Projects:  make(map[string]*Project),
		Functions: make(map[string]*ast.FnDecl),
		Seqs:      make(map[string]*ast.SeqDecl),
		Pars:      make(map[string]*ast.ParDecl),
		Vars:      make(map[string]string),
	}

	for _, prog := range programs {
		for _, stmt := range prog.Statements {
			switch s := stmt.(type) {
			case *ast.ShellDecl:
				if err := mergeShellDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.SanctuaryDecl:
				if err := mergeSanctuaryDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.VarDecl:
				if err := mergeVarDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.ProjectDecl:
				if err := mergeProjectDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.FnDecl:
				if err := mergeFnDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.SeqDecl:
				if err := mergeSeqDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.ParDecl:
				if err := mergeParDecl(cfg, s); err != nil {
					return nil, err
				}
			case *ast.ImportDecl:
				// already handled in parseRecursive, skip
			}
		}
	}

	return cfg, nil
}
