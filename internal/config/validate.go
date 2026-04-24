package config

import (
	"fmt"
	"maps"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

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
