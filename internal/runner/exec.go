package runner

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

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
	fmt.Println(strings.Join(args, ""))
	return nil
}

func (ctx *execContext) execExec(s *ast.ExecStmt) error {
	args, err := ctx.resolveArgs(s.Args)
	if err != nil {
		return err
	}
	cmdStr := strings.Join(args, "")
	fmt.Printf("  exec  %s\n", cmdStr)

	cmd := exec.Command(ctx.cfg.Shell, "-c", cmdStr)
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
	case *ast.BacktickLit:
		if s.VarType == "shell" {
			cmd := exec.Command(ctx.cfg.Shell, "-c", v.Value)
			cmd.Dir = ctx.workDir
			cmd.Env = ctx.buildEnv()
			out, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("shell execution failed for var %s: %w", s.Name, err)
			}
			ctx.vars[s.Name] = strings.TrimRight(string(out), "\n")
		} else {
			ctx.vars[s.Name] = v.Value
		}
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
		case *ast.BacktickLit:
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
