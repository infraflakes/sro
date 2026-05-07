package cmd

import (
	"fmt"

	"github.com/infraflakes/sro/internal/dsl/ast"
)

func labelForStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.FnCall:
		return fmt.Sprintf("%s(pr.%s)", s.FnName, s.ProjectName)
	case *ast.SeqRef:
		return fmt.Sprintf("seq.%s", s.SeqName)
	default:
		return "unknown"
	}
}
