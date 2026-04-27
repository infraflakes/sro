package runner

import (
	"bytes"
	"fmt"
	"io"
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
	writer   io.Writer
}

func newExecContext(cfg *config.Config, proj *config.Project, writer io.Writer) *execContext {
	vars := make(map[string]string, len(cfg.Vars))
	maps.Copy(vars, cfg.Vars)

	baseDir := filepath.Join(cfg.Sanctuary, proj.Dir)

	return &execContext{
		cfg:      cfg,
		project:  proj,
		vars:     vars,
		envStack: []map[string]string{},
		workDir:  baseDir,
		writer:   writer,
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

// indent returns the indentation prefix for the current env nesting depth.
func (ctx *execContext) indent() string {
	return strings.Repeat("  ", len(ctx.envStack))
}

// stdoutIndent returns the indentation prefix for stdout (one level deeper than primitives).
func (ctx *execContext) stdoutIndent() string {
	return strings.Repeat("  ", len(ctx.envStack)+1)
}

// indentWriter wraps an io.Writer and prepends a prefix to each line.
type indentWriter struct {
	w      io.Writer
	prefix string
	atBOL  bool // at beginning of line
}

func newIndentWriter(w io.Writer, prefix string) *indentWriter {
	return &indentWriter{w: w, prefix: prefix, atBOL: true}
}

func (iw *indentWriter) Write(p []byte) (n int, err error) {
	written := 0
	for len(p) > 0 {
		if iw.atBOL && len(p) > 0 {
			// Write indent prefix at the start of each line
			if _, err := iw.w.Write([]byte(iw.prefix)); err != nil {
				return written, err
			}
			iw.atBOL = false
		}

		// Find the next newline
		idx := bytes.IndexByte(p, '\n')
		if idx < 0 {
			// No newline — write the rest
			n, err := iw.w.Write(p)
			written += n
			return written, err
		}

		// Write up to and including the newline
		n, err := iw.w.Write(p[:idx+1])
		written += n
		if err != nil {
			return written, err
		}
		p = p[idx+1:]
		iw.atBOL = true
	}
	return written, nil
}
