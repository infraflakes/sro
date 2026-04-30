package config

import "github.com/infraflakes/sro/internal/dsl/ast"

type Config struct {
	Shell     string
	Sanctuary string
	Projects  map[string]*Project
	Functions map[string]*ast.FnDecl
	Seqs      map[string]*ast.SeqDecl
	Pars      map[string]*ast.ParDecl
	Vars      map[string]string
}

type Project struct {
	Name   string
	URL    string
	Dir    string
	Sync   string // "clone" (default) or "ignore"
	Use    string // optional, path to .sro file inside the project repo
	Branch string // optional, branch to clone
}
