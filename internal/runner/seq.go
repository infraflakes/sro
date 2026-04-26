package runner

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

func (r *Runner) RunSeq(name string) error {
	seq, ok := r.cfg.Seqs[name]
	if !ok {
		return fmt.Errorf("unknown seq: %s", name)
	}
	fmt.Printf("seq %s\n", seq.Name)
	return r.executeSeq(seq)
}

func (r *Runner) executeSeq(seq *ast.SeqDecl) error {
	for _, stmt := range seq.Stmts {
		switch s := stmt.(type) {
		case *ast.FnCall:
			if err := r.executeFnCall(s); err != nil {
				return err
			}
		case *ast.SeqRef:
			refSeq, ok := r.cfg.Seqs[s.SeqName]
			if !ok {
				return fmt.Errorf("unknown seq: %s", s.SeqName)
			}
			if err := r.executeSeq(refSeq); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown seq statement: %T", stmt)
		}
	}
	return nil
}
