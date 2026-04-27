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
	if !r.SuppressHeaders {
		_, _ = fmt.Fprintf(writer, "seq %s\n", seq.Name)
	}
	return r.executeSeqWithWriter(seq, writer)
}

func (r *Runner) executeSeqWithWriter(seq *ast.SeqDecl, writer io.Writer) error {
	for _, stmt := range seq.Stmts {
		switch s := stmt.(type) {
		case *ast.FnCall:
			// Create a new runner to avoid data race on Writer field
			r := r.clone()
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
