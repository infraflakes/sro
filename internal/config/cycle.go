package config

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

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
