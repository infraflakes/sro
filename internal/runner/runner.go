package runner

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
)

type Runner struct {
	cfg             *config.Config
	Writer          io.Writer
	Ctx             context.Context
	SuppressHeaders bool
}

func New(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg, Writer: os.Stdout, Ctx: context.Background(), SuppressHeaders: false}
}

func NewWithContext(cfg *config.Config, ctx context.Context) *Runner {
	return &Runner{cfg: cfg, Writer: os.Stdout, Ctx: ctx, SuppressHeaders: false}
}

func (r *Runner) ExecuteFnCall(call *ast.FnCall) error {
	fn, ok := r.cfg.Functions[call.FnName]
	if !ok {
		return fmt.Errorf("unknown function: %s", call.FnName)
	}
	proj, ok := r.cfg.Projects[call.ProjectName]
	if !ok {
		return fmt.Errorf("unknown project: %s", call.ProjectName)
	}

	writer := r.Writer
	if writer == nil {
		writer = os.Stdout
	}
	if !r.SuppressHeaders {
		fmt.Fprintf(writer, "\033[38;2;91;156;246m%s\033[0m(pr.%s)\n", call.FnName, call.ProjectName)
	}
	ctx := newExecContext(r.cfg, proj, writer)
	return ctx.execFnBody(fn.Body)
}

func (r *Runner) executeFnCall(call *ast.FnCall) error {
	return r.ExecuteFnCall(call)
}
