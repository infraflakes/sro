package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v3/vt"
	"github.com/infraflakes/sro/internal/config"
	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/runner"
	"github.com/infraflakes/sro/internal/tui"
	"github.com/spf13/cobra"
)

var seqCmd = &cobra.Command{
	Use:   "seq <name>",
	Short: "Run a sequential block",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runSeq(args[0])
	},
}

func runSeq(name string) {
	cfg := loadConfig()
	if err := config.ResolveUse(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	seq, ok := cfg.Seqs[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown seq: %s\n", name)
		os.Exit(1)
	}

	model := &tui.Model{
		Type:         "seq",
		Name:         name,
		Status:       "running",
		Selected:     0,
		ScrollOffset: 0,
	}

	// Create a vterm per task in the seq
	for _, stmt := range seq.Stmts {
		label := labelForStmt(stmt)
		vterm := vt.NewMockTerm(vt.MockOptSize(vt.Coord{X: 120, Y: 100}), vt.MockOptColors(1<<24))
		if err := vterm.Start(); err != nil {
			// Still append task with nil VTerm to maintain index alignment
			model.Tasks = append(model.Tasks, tui.Task{
				Label:    label,
				Status:   "failed",
				Expanded: false,
				VTerm:    nil,
			})
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

	// Run seq in background goroutine
	ctx, cancel := context.WithCancel(context.Background())
	model.CancelFunc = cancel
	defer cancel()

	go func() {
		for i, stmt := range seq.Stmts {
			// Skip if vterm failed to start
			if model.Tasks[i].VTerm == nil {
				model.Mu.Lock()
				model.Status = "failed"
				model.Mu.Unlock()
				return
			}

			model.Mu.Lock()
			model.Tasks[i].Status = "running"
			model.Tasks[i].Expanded = true
			// collapse previous
			if i > 0 {
				model.Tasks[i-1].Expanded = false
			}
			model.Mu.Unlock()

			r := runner.NewWithContext(cfg, ctx)
			r.Writer = tui.NewLineCountingWriter(model.Tasks[i].VTerm, &model.Tasks[i].TotalLines)
			r.SuppressHeaders = true

			var err error
			switch s := stmt.(type) {
			case *ast.FnCall:
				err = r.ExecuteFnCall(s)
			case *ast.SeqRef:
				err = r.RunSeqWithWriter(s.SeqName, tui.NewLineCountingWriter(model.Tasks[i].VTerm, &model.Tasks[i].TotalLines))
			}

			if err != nil {
				model.Mu.Lock()
				model.Tasks[i].Status = "failed"
				model.Status = "failed"
				model.Mu.Unlock()
				// mark remaining as pending (fail-fast)
				return
			}
			model.Mu.Lock()
			model.Tasks[i].Status = "ok"
			model.Mu.Unlock()
		}
		model.Mu.Lock()
		model.Status = "ok"
		model.Mu.Unlock()
	}()

	if err := tui.RunWithContext(ctx, model); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
