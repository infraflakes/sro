package config

import (
	"fmt"
	"os"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

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
			case *ast.SanctuaryDecl:
				if cfg.Sanctuary != "" {
					return nil, fmt.Errorf("duplicate sanctuary declaration")
				}
				cfg.Sanctuary = os.ExpandEnv(s.Value)

			case *ast.VarDecl:
				name := s.Name
				if _, exists := cfg.Vars[name]; exists {
					return nil, fmt.Errorf("duplicate variable: %s", name)
				}
				switch v := s.Value.(type) {
				case *ast.StringLit:
					cfg.Vars[name] = v.Value
				case *ast.VarRef:
					resolved, ok := cfg.Vars[v.Name]
					if !ok {
						return nil, fmt.Errorf("undefined variable: $%s", v.Name)
					}
					cfg.Vars[name] = resolved
				}

			case *ast.ProjectDecl:
				if _, exists := cfg.Projects[s.Name]; exists {
					return nil, fmt.Errorf("duplicate project: %s", s.Name)
				}
				proj := &Project{Name: s.Name, Sync: "clone"} // default
				for _, f := range s.Fields {
					switch f.Key {
					case "url":
						proj.URL = f.Value
					case "dir":
						proj.Dir = f.Value
					case "sync":
						proj.Sync = f.Value
					case "use":
						proj.Use = f.Value
					}
				}
				cfg.Projects[s.Name] = proj

			case *ast.FnDecl:
				if _, exists := cfg.Functions[s.Name]; exists {
					return nil, fmt.Errorf("duplicate function: %s", s.Name)
				}
				cfg.Functions[s.Name] = s

			case *ast.SeqDecl:
				if _, exists := cfg.Seqs[s.Name]; exists {
					return nil, fmt.Errorf("duplicate seq: %s", s.Name)
				}
				cfg.Seqs[s.Name] = s

			case *ast.ParDecl:
				if _, exists := cfg.Pars[s.Name]; exists {
					return nil, fmt.Errorf("duplicate par: %s", s.Name)
				}
				cfg.Pars[s.Name] = s

			case *ast.ImportDecl:
				// already handled in parseRecursive, skip
			}
		}
	}

	return cfg, nil
}
