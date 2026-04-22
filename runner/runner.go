package runner

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/infraflakes/sro/ast"
	"github.com/infraflakes/sro/config"
)

type Runner struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg}
}

type execContext struct {
	cfg      *config.Config
	project  *config.Project
	vars     map[string]string
	envStack []map[string]string
	workDir  string
}

func newExecContext(cfg *config.Config, proj *config.Project) *execContext {
	vars := make(map[string]string, len(cfg.Vars))
	maps.Copy(vars, cfg.Vars)

	baseDir := filepath.Join(cfg.Sanctuary, proj.Dir)

	return &execContext{
		cfg:      cfg,
		project:  proj,
		vars:     vars,
		envStack: []map[string]string{},
		workDir:  baseDir,
	}
}

func (ctx *execContext) resolveArgs(args []ast.Expr) ([]string, error) {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		switch a := arg.(type) {
		case *ast.StringLit:
			result = append(result, a.Value)
		case *ast.VarRef:
			val, ok := ctx.vars[a.Name]
			if !ok {
				line, col := a.Pos()
				return nil, fmt.Errorf("%d:%d: undefined variable $%s", line, col, a.Name)
			}
			result = append(result, val)
		default:
			return nil, fmt.Errorf("unexpected expression type: %T", arg)
		}
	}
	return result, nil
}

func (ctx *execContext) buildEnv() []string {
	env := map[string]string{}
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	for _, layer := range ctx.envStack {
		maps.Copy(env, layer)
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

func (ctx *execContext) execFnBody(body []ast.FnStmt) error {
	for _, stmt := range body {
		switch s := stmt.(type) {
		case *ast.LogStmt:
			if err := ctx.execLog(s); err != nil {
				return err
			}
		case *ast.ExecStmt:
			if err := ctx.execExec(s); err != nil {
				return err
			}
		case *ast.CdStmt:
			if err := ctx.execCd(s); err != nil {
				return err
			}
		case *ast.VarDecl:
			if err := ctx.execVarDecl(s); err != nil {
				return err
			}
		case *ast.EnvBlock:
			if err := ctx.execEnvBlock(s); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown fn statement: %T", stmt)
		}
	}
	return nil
}

func (ctx *execContext) execLog(s *ast.LogStmt) error {
	args, err := ctx.resolveArgs(s.Args)
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(args, " "))
	return nil
}

func (ctx *execContext) execExec(s *ast.ExecStmt) error {
	args, err := ctx.resolveArgs(s.Args)
	if err != nil {
		return err
	}
	cmdStr := strings.Join(args, " ")
	fmt.Printf("  exec  %s\n", cmdStr)

	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = ctx.workDir
	cmd.Env = ctx.buildEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("exec failed with exit code %d: %s", exitErr.ExitCode(), cmdStr)
		}
		return fmt.Errorf("exec failed: %s: %w", cmdStr, err)
	}
	return nil
}

func (ctx *execContext) execCd(s *ast.CdStmt) error {
	baseDir := filepath.Join(ctx.cfg.Sanctuary, ctx.project.Dir)
	if s.Arg == "." {
		ctx.workDir = baseDir
	} else {
		ctx.workDir = filepath.Join(baseDir, s.Arg)
	}
	if _, err := os.Stat(ctx.workDir); err != nil {
		return fmt.Errorf("cd %q: %w", s.Arg, err)
	}
	return nil
}

func (ctx *execContext) execVarDecl(s *ast.VarDecl) error {
	switch v := s.Value.(type) {
	case *ast.StringLit:
		ctx.vars[s.Name] = v.Value
	case *ast.VarRef:
		val, ok := ctx.vars[v.Name]
		if !ok {
			return fmt.Errorf("undefined variable: $%s", v.Name)
		}
		ctx.vars[s.Name] = val
	default:
		return fmt.Errorf("unexpected value type: %T", v)
	}
	return nil
}

func (ctx *execContext) execEnvBlock(s *ast.EnvBlock) error {
	layer := make(map[string]string, len(s.Pairs))
	for _, p := range s.Pairs {
		switch v := p.Value.(type) {
		case *ast.StringLit:
			layer[p.Key] = v.Value
		case *ast.VarRef:
			val, ok := ctx.vars[v.Name]
			if !ok {
				return fmt.Errorf("undefined variable: $%s", v.Name)
			}
			layer[p.Key] = val
		}
	}
	ctx.envStack = append(ctx.envStack, layer)

	// Snapshot vars before body so local vars don't leak
	savedVars := make(map[string]string, len(ctx.vars))
	maps.Copy(savedVars, ctx.vars)

	err := ctx.execFnBody(s.Body)

	ctx.vars = savedVars
	ctx.envStack = ctx.envStack[:len(ctx.envStack)-1]
	return err
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
