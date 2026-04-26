package cmd

import (
	"bytes"
	"fmt"
	"os"

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
		Type:     "seq",
		Name:     name,
		Status:   "running",
		Selected: 0,
	}

	// Create a buffer per task in the seq
	for _, stmt := range seq.Stmts {
		label := labelForStmt(stmt)
		buf := &bytes.Buffer{}
		model.Tasks = append(model.Tasks, tui.Task{
			Label:    label,
			Status:   "pending",
			Expanded: false,
			Output:   buf,
			Writer:   buf,
		})
	}

	// Run seq in background goroutine
	go func() {
		for i, stmt := range seq.Stmts {
			model.Tasks[i].Status = "running"
			model.Tasks[i].Expanded = true
			// collapse previous
			if i > 0 {
				model.Tasks[i-1].Expanded = false
			}

			r := runner.New(cfg)
			r.Writer = model.Tasks[i].Writer

			var err error
			switch s := stmt.(type) {
			case *ast.FnCall:
				err = r.ExecuteFnCall(s)
			case *ast.SeqRef:
				err = r.RunSeqWithWriter(s.SeqName, model.Tasks[i].Writer)
			}

			if err != nil {
				model.Tasks[i].Status = "failed"
				model.Status = "failed"
				// mark remaining as pending (fail-fast)
				return
			}
			model.Tasks[i].Status = "ok"
		}
		model.Status = "ok"
	}()

	if err := tui.Run(model); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
