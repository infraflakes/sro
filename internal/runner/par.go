package runner

import (
	"fmt"
	"strings"
	"sync"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

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
