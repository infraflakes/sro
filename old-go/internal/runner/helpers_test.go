package runner

import (
	"testing"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func newTestToken(typ token.TokenType) token.Token {
	return token.Token{
		Type:    typ,
		Literal: string(typ),
		Line:    1,
		Col:     1,
	}
}

func newBacktickLit(value string) *ast.BacktickLit {
	return &ast.BacktickLit{
		Token: newTestToken(token.BACKTICK),
		Parts: []ast.TemplatePart{{IsVar: false, Value: value}},
	}
}

func newVarRef(name string) *ast.VarRef {
	return &ast.VarRef{
		Token: newTestToken(token.IDENT),
		Name:  name,
	}
}

func newLogStmt(value ast.Expr) *ast.LogStmt {
	return &ast.LogStmt{
		Token: newTestToken(token.LOG),
		Value: value,
	}
}

func newExecStmt(value ast.Expr) *ast.ExecStmt {
	return &ast.ExecStmt{
		Token: newTestToken(token.EXEC),
		Value: value,
	}
}

func newCdStmt(arg string) *ast.CdStmt {
	return &ast.CdStmt{
		Token: newTestToken(token.CD),
		Arg:   arg,
	}
}

func newVarDecl(name string, varType string, value ast.Expr) *ast.VarDecl {
	return &ast.VarDecl{
		Token:   newTestToken(token.VAR),
		VarType: varType,
		Name:    name,
		Value:   value,
	}
}

func newEnvBlock(pairs []ast.EnvPair, body []ast.FnStmt) *ast.EnvBlock {
	return &ast.EnvBlock{
		Token: newTestToken(token.ENV),
		Pairs: pairs,
		Body:  body,
	}
}

func newFnDecl(name string, body []ast.FnStmt) *ast.FnDecl {
	return &ast.FnDecl{
		Token: newTestToken(token.FN),
		Name:  name,
		Body:  body,
	}
}

func newFnCall(fnName, projectName string) *ast.FnCall {
	return &ast.FnCall{
		Token:       newTestToken(token.IDENT),
		FnName:      fnName,
		ProjectName: projectName,
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Shell:     "bash",
		Sanctuary: t.TempDir(),
		Projects: map[string]*config.Project{
			"testproj": {
				Name: "testproj",
				URL:  "http://example.com",
				Dir:  "testproj",
				Sync: "clone",
			},
		},
		Functions: make(map[string]*ast.FnDecl),
		Seqs:      make(map[string]*ast.SeqDecl),
		Pars:      make(map[string]*ast.ParDecl),
		Vars:      map[string]string{},
	}
}
