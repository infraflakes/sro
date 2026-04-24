package runner

import (
	"fmt"
	"strings"
	"sync"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/config"
)

type Runner struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg}
}

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

func (r *Runner) executeFnCall(call *ast.FnCall) error {
	fn, ok := r.cfg.Functions[call.FnName]
	if !ok {
		return fmt.Errorf("unknown function: %s", call.FnName)
	}
	proj, ok := r.cfg.Projects[call.ProjectName]
	if !ok {
		return fmt.Errorf("unknown project: %s", call.ProjectName)
	}

	fmt.Printf("  %s(pr.%s)\n", call.FnName, call.ProjectName)
	ctx := newExecContext(r.cfg, proj)
	return ctx.execFnBody(fn.Body)
}

func (r *Runner) RunPar(name string) error {
	par, ok := r.cfg.Pars[name]
	if !ok {
		return fmt.Errorf("unknown par: %s", name)
	}

	fmt.Printf("par %s\n", par.Name)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, stmt := range par.Stmts {
		switch s := stmt.(type) {
		case *ast.FnCall:
			wg.Add(1)
			go func(call *ast.FnCall) {
				defer wg.Done()
				if err := r.executeFnCall(call); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("%s(pr.%s): %w", call.FnName, call.ProjectName, err))
					mu.Unlock()
				}
			}(s)
		case *ast.SeqRef:
			wg.Add(1)
			go func(ref *ast.SeqRef) {
				defer wg.Done()
				refSeq, ok := r.cfg.Seqs[ref.SeqName]
				if !ok {
					mu.Lock()
					errs = append(errs, fmt.Errorf("unknown seq: %s", ref.SeqName))
					mu.Unlock()
					return
				}
				if err := r.executeSeq(refSeq); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("seq.%s: %w", ref.SeqName, err))
					mu.Unlock()
				}
			}(s)
		default:
			mu.Lock()
			errs = append(errs, fmt.Errorf("unknown par statement: %T", stmt))
			mu.Unlock()
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("par %s: %d error(s):\n%s", name, len(errs), strings.Join(msgs, "\n"))
	}
	return nil
}
