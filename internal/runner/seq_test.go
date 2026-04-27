package runner

import (
	"bytes"
	"strings"
	"testing"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

func TestSeqFailFast(t *testing.T) {
	cfg := testConfig(t)

	// R10: verify second fn wasn't called in fail-fast
	// Function that will fail
	failFn := newFnDecl("fail", []ast.FnStmt{
		newExecStmt(newBacktickLit("false")),
	})
	cfg.Functions["fail"] = failFn

	// Function that should NOT run - use a log to track if it was called
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

	var buf bytes.Buffer
	r := New(cfg)
	r.Writer = &buf
	err := r.RunSeq("testseq")
	if err == nil {
		t.Fatal("expected error from failing exec")
	}
	// Verify the second function was NOT called by checking its log is absent
	output := buf.String()
	if strings.Contains(output, "second-called") {
		t.Fatal("second function ran despite fail-fast — seq did not stop after first failure")
	}
}

func TestSeqCallsSeq(t *testing.T) {
	cfg := testConfig(t)
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
	cfg := testConfig(t)
	r := New(cfg)
	err := r.RunSeq("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown seq") {
		t.Fatalf("expected unknown seq error, got: %v", err)
	}
}

func TestUnknownFunctionInSeq(t *testing.T) {
	cfg := testConfig(t)
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
	cfg := testConfig(t)
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
