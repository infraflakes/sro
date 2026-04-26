package runner

import (
	"fmt"
	"io"
	"os"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

func (r *Runner) RunSeq(name string) error {
	return r.RunSeqWithWriter(name, r.Writer)
}

func (r *Runner) RunSeqWithWriter(name string, writer io.Writer) error {
	seq, ok := r.cfg.Seqs[name]
	if !ok {
		return fmt.Errorf("unknown seq: %s", name)
	}
	if writer == nil {
		writer = os.Stdout
	}
	fmt.Fprintf(writer, "seq %s\n", seq.Name)
	return r.executeSeqWithWriter(seq, writer)
}

func (r *Runner) executeSeq(seq *ast.SeqDecl) error {
	return r.executeSeqWithWriter(seq, r.Writer)
}

func (r *Runner) executeSeqWithWriter(seq *ast.SeqDecl, writer io.Writer) error {
	for _, stmt := range seq.Stmts {
		switch s := stmt.(type) {
		case *ast.FnCall:
			r.Writer = writer
			if err := r.executeFnCall(s); err != nil {
				return err
			}
		case *ast.SeqRef:
			refSeq, ok := r.cfg.Seqs[s.SeqName]
			if !ok {
				return fmt.Errorf("unknown seq: %s", s.SeqName)
			}
			if err := r.executeSeqWithWriter(refSeq, writer); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown seq statement: %T", stmt)
		}
	}
	return nil
}
