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
