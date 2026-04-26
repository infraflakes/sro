package runner

import (
	"fmt"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
)

type Runner struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg}
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
