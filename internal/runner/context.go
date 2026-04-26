package runner

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
)

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

func (ctx *execContext) resolveExpr(expr ast.Expr) (string, error) {
	switch e := expr.(type) {
	case *ast.BacktickLit:
		var sb strings.Builder
		for _, part := range e.Parts {
			if part.IsVar {
				val, ok := ctx.vars[part.Value]
				if !ok {
					line, col := e.Pos()
					return "", fmt.Errorf("%d:%d: undefined variable ${%s}", line, col, part.Value)
				}
				sb.WriteString(val)
			} else {
				sb.WriteString(part.Value)
			}
		}
		return sb.String(), nil
	case *ast.VarRef:
		val, ok := ctx.vars[e.Name]
		if !ok {
			line, col := e.Pos()
			return "", fmt.Errorf("%d:%d: undefined variable $%s", line, col, e.Name)
		}
		return val, nil
	default:
		return "", fmt.Errorf("unexpected expression type: %T", expr)
	}
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
