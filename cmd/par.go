package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gdamore/tcell/v3/vt"
	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/runner"
	"github.com/infraflakes/sro/internal/tui"
	"github.com/spf13/cobra"
)

var parCmd = &cobra.Command{
	Use:   "par <name>",
	Short: "Run a parallel block",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runPar(args[0])
	},
}

func runPar(name string) {
	cfg := loadConfig()
	if err := config.ResolveUse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	par, ok := cfg.Pars[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown par: %s\n", name)
		os.Exit(1)
	}

	// Fallback to plain stdout if --no-tui is set
	if noTui {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		r := runner.NewWithContext(cfg, ctx)
		if err := r.RunPar(name); err != nil {
			fmt.Fprintf(os.Stderr, "par error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	model := &tui.Model{
		Type:         "par",
		Name:         name,
		Status:       "running",
		Selected:     0,
		ScrollOffset: 0,
	}

	// Create a vterm per task in the par
	for _, stmt := range par.Stmts {
		label := labelForStmt(stmt)
		vterm := vt.NewMockTerm(vt.MockOptSize(vt.Coord{X: 120, Y: 100}), vt.MockOptColors(1<<24))
		if err := vterm.Start(); err != nil {
			continue
		}
		_, _ = vterm.Write([]byte("\x1b[20h")) // enable newline mode: LF implies CR
		model.Tasks = append(model.Tasks, tui.Task{
			Label:    label,
			Status:   "pending",
			Expanded: false,
			VTerm:    vterm,
		})
	}

	// Run par in background goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var hasFailed bool

		for i, stmt := range par.Stmts {
			wg.Add(1)
			go func(idx int, s ast.Stmt) {
				defer wg.Done()

				model.Tasks[idx].Status = "running"
				// Don't auto-expand in par - user controls expansion

				r := runner.NewWithContext(cfg, ctx)
				r.Writer = tui.NewLineCountingWriter(model.Tasks[idx].VTerm, &model.Tasks[idx].TotalLines)
				r.SuppressHeaders = true

				var err error
				switch stmt := s.(type) {
				case *ast.FnCall:
					err = r.ExecuteFnCall(stmt)
				case *ast.SeqRef:
					err = r.RunSeqWithWriter(stmt.SeqName, tui.NewLineCountingWriter(model.Tasks[idx].VTerm, &model.Tasks[idx].TotalLines))
				}

				if err != nil {
					model.Tasks[idx].Status = "failed"
					mu.Lock()
					hasFailed = true
					mu.Unlock()
				} else {
					model.Tasks[idx].Status = "ok"
				}
			}(i, stmt)
		}

		wg.Wait()

		mu.Lock()
		if hasFailed {
			model.Status = "failed"
		} else {
			model.Status = "ok"
		}
		mu.Unlock()
	}()

	if err := tui.RunWithContext(ctx, model); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
