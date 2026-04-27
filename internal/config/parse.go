package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/lexer"
	"github.com/infraflakes/sro/internal/dsl/parser"
)

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
		errStrs := make([]string, len(p.Errors()))
		for i, err := range p.Errors() {
			errStrs[i] = err.Error()
		}
		return nil, fmt.Errorf("parse errors in %s:\n%s", absPath, strings.Join(errStrs, "\n"))
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
