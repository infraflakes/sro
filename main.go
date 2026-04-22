package main

import (
	"fmt"

	"os"

	"github.com/infraflakes/sro/ast"
	"github.com/infraflakes/sro/lexer"
	"github.com/infraflakes/sro/parser"
)

func main() {
	data, err := os.ReadFile("examples/main.sro")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}
	input := string(data)

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}

	var sanctuary, importCnt, vars, project, fn, seq, par int
	for _, stmt := range prog.Statements {
		switch stmt.(type) {
		case *ast.SanctuaryDecl:
			sanctuary++
		case *ast.ImportDecl:
			importCnt++
		case *ast.VarDecl:
			vars++
		case *ast.ProjectDecl:
			project++
		case *ast.FnDecl:
			fn++
		case *ast.SeqDecl:
			seq++
		case *ast.ParDecl:
			par++
		default:
			fmt.Fprintf(os.Stderr, "unknown statement type: %T\n", stmt)
			os.Exit(1)
		}
	}

	fmt.Printf("%d sanctuary, %d import, %d vars, %d project, %d fn, %d seq, %d par\n",
		sanctuary, importCnt, vars, project, fn, seq, par)
}
