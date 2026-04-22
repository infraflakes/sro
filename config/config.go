package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/infraflakes/sro/ast"
	"github.com/infraflakes/sro/lexer"
	"github.com/infraflakes/sro/parser"
)

type Config struct {
	Sanctuary string
	Projects  map[string]*Project
	Functions map[string]*ast.FnDecl
	Seqs      map[string]*ast.SeqDecl
	Pars      map[string]*ast.ParDecl
	Vars      map[string]string
}

type Project struct {
	Name string
	URL  string
	Dir  string
	Sync string // "clone" (default) or "ignore"
	Use  string // optional, path to .sro file inside the project repo
}

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

// ResolveUse resolves `use` fields in project declarations.
// For each project where Use != "", it parses the use file and merges
// fn/seq/par/var declarations into cfg, then runs full validation.
func ResolveUse(cfg *Config) error {
	for _, proj := range cfg.Projects {
		if proj.Use == "" {
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

func parseRecursive(filePath string, visited map[string]bool) ([]*ast.Program, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular import detected: %s", absPath)
	}
	visited[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", absPath, err)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s:\n%s", absPath, strings.Join(p.Errors(), "\n"))
	}

	results := []*ast.Program{}

	// Process imports first (depth-first)
	baseDir := filepath.Dir(absPath)
	for _, stmt := range prog.Statements {
		imp, ok := stmt.(*ast.ImportDecl)
		if !ok {
			continue
		}
		for _, relPath := range imp.Paths {
			importAbs := filepath.Join(baseDir, relPath)
			imported, err := parseRecursive(importAbs, visited)
			if err != nil {
				return nil, err
			}
			results = append(results, imported...)
		}
	}

	// Then add the current program
	results = append(results, prog)
	return results, nil
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

func validateFnBody(stmts []ast.FnStmt, localVars map[string]bool, fnName string) error {
	for i, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.VarDecl:
			// Check that var refs in the value are valid
			if err := validateExpr(s.Value, localVars, fnName, i); err != nil {
				return err
			}
			// Add this var to local scope for subsequent statements
			localVars[s.Name] = true
		case *ast.LogStmt:
			for _, arg := range s.Args {
				if err := validateExpr(arg, localVars, fnName, i); err != nil {
					return err
				}
			}
		case *ast.ExecStmt:
			for _, arg := range s.Args {
				if err := validateExpr(arg, localVars, fnName, i); err != nil {
					return err
				}
			}
		case *ast.EnvBlock:
			// Snapshot localVars before entering env block
			savedVars := make(map[string]bool, len(localVars))
			maps.Copy(savedVars, localVars)
			// Validate env pairs (now Expr, can have var refs)
			for _, p := range s.Pairs {
				if err := validateExpr(p.Value, localVars, fnName, i); err != nil {
					return err
				}
			}
			// Validate body with current localVars (which may be extended)
			if err := validateFnBody(s.Body, localVars, fnName); err != nil {
				return err
			}
			// Restore localVars after env block
			localVars = savedVars
		case *ast.CdStmt:
			// CdStmt.Arg is string, no var refs
		}
	}
	return nil
}

func detectSeqCycles(cfg *Config) []string {
	var errs []string
	visited := map[string]bool{}
	inStack := map[string]bool{}

	var dfs func(name string) bool
	dfs = func(name string) bool {
		if inStack[name] {
			errs = append(errs, fmt.Sprintf("seq/par cycle detected: %s", name))
			return true
		}
		if visited[name] {
			return false
		}
		visited[name] = true
		inStack[name] = true

		// Check seqs
		if seq, ok := cfg.Seqs[name]; ok {
			for _, stmt := range seq.Stmts {
				if ref, ok := stmt.(*ast.SeqRef); ok {
					if dfs(ref.SeqName) {
						return true
					}
				}
			}
		}

		// Check pars
		if par, ok := cfg.Pars[name]; ok {
			for _, stmt := range par.Stmts {
				if ref, ok := stmt.(*ast.SeqRef); ok {
					if dfs(ref.SeqName) {
						return true
					}
				}
			}
		}

		inStack[name] = false
		return false
	}

	for name := range cfg.Seqs {
		dfs(name)
	}
	for name := range cfg.Pars {
		dfs(name)
	}
	return errs
}

func validateExpr(expr ast.Expr, localVars map[string]bool, fnName string, stmtIndex int) error {
	switch e := expr.(type) {
	case *ast.VarRef:
		if !localVars[e.Name] {
			return fmt.Errorf("function %q stmt %d: undefined variable $%s", fnName, stmtIndex, e.Name)
		}
	case *ast.StringLit:
		// Always valid
	}
	return nil
}

func validateBase(cfg *Config) error {
	var errs []string

	if cfg.Sanctuary == "" {
		errs = append(errs, "sanctuary is required")
	}

	for _, proj := range cfg.Projects {
		if proj.URL == "" {
			errs = append(errs, fmt.Sprintf("project %q: url is required", proj.Name))
		}
		if proj.Dir == "" {
			errs = append(errs, fmt.Sprintf("project %q: dir is required", proj.Name))
		}
		if proj.Sync != "clone" && proj.Sync != "ignore" {
			errs = append(errs, fmt.Sprintf("project %q: sync must be \"clone\" or \"ignore\", got %q", proj.Name, proj.Sync))
		}
	}

	dirs := map[string]string{}
	for _, proj := range cfg.Projects {
		if other, exists := dirs[proj.Dir]; exists {
			errs = append(errs, fmt.Sprintf("project %q and %q have the same dir %q", proj.Name, other, proj.Dir))
		}
		dirs[proj.Dir] = proj.Name
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func validateFull(cfg *Config) error {
	var errs []string

	// Check seq/par references
	for name, seq := range cfg.Seqs {
		for _, stmt := range seq.Stmts {
			switch s := stmt.(type) {
			case *ast.FnCall:
				if _, ok := cfg.Functions[s.FnName]; !ok {
					errs = append(errs, fmt.Errorf("seq %q: unknown function %q", name, s.FnName).Error())
				}
				if _, ok := cfg.Projects[s.ProjectName]; !ok {
					errs = append(errs, fmt.Errorf("seq %q: unknown project %q", name, s.ProjectName).Error())
				}
			case *ast.SeqRef:
				if _, ok := cfg.Seqs[s.SeqName]; !ok {
					errs = append(errs, fmt.Errorf("seq %q: unknown seq %q", name, s.SeqName).Error())
				}
			}
		}
	}

	for name, par := range cfg.Pars {
		for _, stmt := range par.Stmts {
			switch s := stmt.(type) {
			case *ast.FnCall:
				if _, ok := cfg.Functions[s.FnName]; !ok {
					errs = append(errs, fmt.Errorf("par %q: unknown function %q", name, s.FnName).Error())
				}
				if _, ok := cfg.Projects[s.ProjectName]; !ok {
					errs = append(errs, fmt.Errorf("par %q: unknown project %q", name, s.ProjectName).Error())
				}
			case *ast.SeqRef:
				if _, ok := cfg.Seqs[s.SeqName]; !ok {
					errs = append(errs, fmt.Errorf("par %q: unknown seq %q", name, s.SeqName).Error())
				}
			}
		}
	}

	// Validate var references in function bodies
	for name, fn := range cfg.Functions {
		localVars := map[string]bool{}
		for k := range cfg.Vars {
			localVars[k] = true
		}
		if err := validateFnBody(fn.Body, localVars, name); err != nil {
			errs = append(errs, err.Error())
		}
	}

	// Detect seq/par cycles
	cycleErrs := detectSeqCycles(cfg)
	errs = append(errs, cycleErrs...)

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}
