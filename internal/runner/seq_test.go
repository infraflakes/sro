package runner

import (
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func TestSeqFailFast(t *testing.T) {
	cfg := testConfig()

	// Function that will fail
	failFn := newFnDecl("fail", []ast.FnStmt{
		newExecStmt(newBacktickLit("false")),
	})
	cfg.Functions["fail"] = failFn

	// Function that should NOT run
	secondFn := newFnDecl("second", []ast.FnStmt{
		newLogStmt(newBacktickLit("second-called")),
	})
	cfg.Functions["second"] = secondFn

	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "testseq",
		Stmts: []ast.SeqStmt{
			newFnCall("fail", "testproj"),
			newFnCall("second", "testproj"),
		},
	}
	cfg.Seqs["testseq"] = seq

	r := New(cfg)
	err := r.RunSeq("testseq")
	if err == nil {
		t.Fatal("expected error from failing exec")
	}
	// The second function should not have been called — can't easily test without side effects
}

func TestSeqCallsSeq(t *testing.T) {
	cfg := testConfig()
	logFn := newFnDecl("logfn", []ast.FnStmt{
		newLogStmt(newBacktickLit("inner-log")),
	})
	cfg.Functions["logfn"] = logFn

	innerSeq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "inner",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
		},
	}
	cfg.Seqs["inner"] = innerSeq

	outerSeq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "outer",
		Stmts: []ast.SeqStmt{
			newFnCall("logfn", "testproj"),
			&ast.SeqRef{
				Token:   newTestToken(token.SEQ),
				SeqName: "inner",
			},
		},
	}
	cfg.Seqs["outer"] = outerSeq

	r := New(cfg)
	// Can't easily capture output here; just ensure no error
	if err := r.RunSeq("outer"); err != nil {
		t.Fatalf("RunSeq error: %v", err)
	}
}

func TestRunSeqUnknown(t *testing.T) {
	cfg := testConfig()
	r := New(cfg)
	err := r.RunSeq("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown seq") {
		t.Fatalf("expected unknown seq error, got: %v", err)
	}
}

func TestUnknownFunctionInSeq(t *testing.T) {
	cfg := testConfig()
	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "badseq",
		Stmts: []ast.SeqStmt{
			newFnCall("nofn", "testproj"),
		},
	}
	cfg.Seqs["badseq"] = seq
	r := New(cfg)
	err := r.RunSeq("badseq")
	if err == nil || !strings.Contains(err.Error(), "unknown function") {
		t.Fatalf("expected unknown function error, got: %v", err)
	}
}

func TestUnknownProjectInSeq(t *testing.T) {
	cfg := testConfig()
	cfg.Functions["dummy"] = newFnDecl("dummy", []ast.FnStmt{})
	seq := &ast.SeqDecl{
		Token: newTestToken(token.SEQ),
		Name:  "badseq",
		Stmts: []ast.SeqStmt{
			newFnCall("dummy", "noproj"),
		},
	}
	cfg.Seqs["badseq"] = seq
	r := New(cfg)
	err := r.RunSeq("badseq")
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("expected unknown project error, got: %v", err)
	}
}
