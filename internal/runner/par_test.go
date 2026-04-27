package runner

import (
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func TestParContinuesOnFailure(t *testing.T) {
	cfg := testConfig(t)

	// Function that fails
	failFn := newFnDecl("fail", []ast.FnStmt{
		newExecStmt(newBacktickLit("false")),
	})
	cfg.Functions["fail"] = failFn

	// Function that succeeds
	successFn := newFnDecl("success", []ast.FnStmt{
		newLogStmt(newBacktickLit("success-called")),
	})
	cfg.Functions["success"] = successFn

	par := &ast.ParDecl{
		Token: newTestToken(token.PAR),
		Name:  "testpar",
		Stmts: []ast.ParStmt{
			newFnCall("fail", "testproj"),
			newFnCall("success", "testproj"),
		},
	}
	cfg.Pars["testpar"] = par

	r := New(cfg)
	err := r.RunPar("testpar")
	if err == nil {
		t.Fatal("expected error from par with failing task")
	}
	if !strings.Contains(err.Error(), "fail(pr.testproj)") {
		t.Logf("error msg: %s", err.Error())
	}
	// success should not appear in errors (it succeeded)
}

func TestParCallsSeq(t *testing.T) {
	cfg := testConfig(t)
	logFn := newFnDecl("logfn", []ast.FnStmt{
		newLogStmt(newBacktickLit("par-seq-log")),
	})
	cfg.Functions["logfn"] = logFn

	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "myseq",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
		},
	}
	cfg.Seqs["myseq"] = seq

	par := &ast.ParDecl{
		Token: newTestToken(token.PAR),
		Name:  "mypar",
		Stmts: []ast.ParStmt{
			&ast.SeqRef{
				Token:   newTestToken(token.SEQ),
				SeqName: "myseq",
			},
		},
	}
	cfg.Pars["mypar"] = par

	r := New(cfg)
	if err := r.RunPar("mypar"); err != nil {
		t.Fatalf("RunPar error: %v", err)
	}
}

func TestRunParUnknown(t *testing.T) {
	cfg := testConfig(t)
	r := New(cfg)
	err := r.RunPar("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown par") {
		t.Fatalf("expected unknown par error, got: %v", err)
	}
}
