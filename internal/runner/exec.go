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
	msg, err := ctx.resolveExpr(s.Value)
	if err != nil {
		return err
	}
	writer := ctx.writer
	if writer == nil {
		writer = os.Stdout
	}
	_, _ = fmt.Fprintf(writer, "%s\033[38;2;255;203;107mlog  %s\033[0m\n", ctx.indent(), msg)
	return nil
}

func (ctx *execContext) execExec(s *ast.ExecStmt) error {
	cmdStr, err := ctx.resolveExpr(s.Value)
	if err != nil {
		return err
	}
	writer := ctx.writer
	if writer == nil {
		writer = os.Stdout
	}
	_, _ = fmt.Fprintf(writer, "%s\033[38;2;91;156;246mexec %s\033[0m\n", ctx.indent(), cmdStr)

	cmd := exec.CommandContext(ctx.ctx, ctx.cfg.Shell, "-c", cmdStr)
	cmd.Dir = ctx.workDir
	cmd.Env = ctx.buildEnv()
	indented := newIndentWriter(writer, ctx.stdoutIndent())
	cmd.Stdout = indented
	cmd.Stderr = indented

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
	writer := ctx.writer
	if writer == nil {
		writer = os.Stdout
	}
	_, _ = fmt.Fprintf(writer, "%s\033[38;2;255;203;107mcd   %s\033[0m\n", ctx.indent(), s.Arg)
	return nil
}

func (ctx *execContext) execVarDecl(s *ast.VarDecl) error {
	val, err := ctx.resolveExpr(s.Value)
	if err != nil {
		return err
	}
	if s.VarType == "shell" {
		cmd := exec.Command(ctx.cfg.Shell, "-c", val)
		cmd.Dir = ctx.workDir
		cmd.Env = ctx.buildEnv()
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("shell execution failed for var %s: %w", s.Name, err)
		}
		ctx.vars[s.Name] = strings.TrimRight(string(out), "\n")
	} else {
		ctx.vars[s.Name] = val
	}
	return nil
}

func (ctx *execContext) execEnvBlock(s *ast.EnvBlock) error {
	layer := make(map[string]string, len(s.Pairs))
	for _, p := range s.Pairs {
		val, err := ctx.resolveExpr(p.Value)
		if err != nil {
			return err
		}
		layer[p.Key] = val
	}

	writer := ctx.writer
	if writer == nil {
		writer = os.Stdout
	}
	if len(s.Pairs) > 0 {
		_, _ = fmt.Fprintf(writer, "%s\033[38;2;199;146;234menv  %s\033[0m\n", ctx.indent(), s.Pairs[0].Key)
	} else {
		_, _ = fmt.Fprintf(writer, "%s\033[38;2;199;146;234menv\033[0m\n", ctx.indent())
	}

	ctx.envStack = append(ctx.envStack, layer)
	ctx.envDirty = true

	// Snapshot vars before body so local vars don't leak
	savedVars := make(map[string]string, len(ctx.vars))
	maps.Copy(savedVars, ctx.vars)

	err := ctx.execFnBody(s.Body)

	ctx.vars = savedVars
	ctx.envStack = ctx.envStack[:len(ctx.envStack)-1]
	ctx.envDirty = true
	return err
}
