package tui

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/gdamore/tcell/v3/vt"
)

type Model struct {
	Type         string // "seq", "par", "sync"
	Name         string
	Status       string // "running", "ok", "failed"
	Tasks        []Task
	Selected     int // keyboard cursor position
	ScrollOffset int // viewport scroll offset
	Mu           sync.Mutex
	CancelFunc   context.CancelFunc // called on TUI exit to cancel background tasks
}

type Task struct {
	Label      string
	Status     string // "ok", "running", "failed", "pending"
	Expanded   bool
	VTerm      vt.MockTerm // virtual terminal for this task
	TotalLines int64       // total lines output (for accurate pruning calculation)
}

// lineCountingWriter wraps an io.Writer to count newline characters
type lineCountingWriter struct {
	writer     io.Writer
	totalLines *int64
}

func (w *lineCountingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	var added int64
	for _, b := range p[:n] {
		if b == '\n' {
			added++
		}
	}
	if added > 0 {
		atomic.AddInt64(w.totalLines, added)
	}
	return n, err
}

// NewLineCountingWriter creates a writer that counts newlines written to it
func NewLineCountingWriter(writer io.Writer, totalLines *int64) io.Writer {
	return &lineCountingWriter{
		writer:     writer,
		totalLines: totalLines,
	}
}

// TaskRenderedHeight returns the total rows a task occupies when rendered.
// This must stay in sync with renderExpandedPanel's layout.
func TaskRenderedHeight(task *Task) int {
	h := 1 // collapsed task row
	if task.Expanded && task.VTerm != nil {
		cursorPos := task.VTerm.Pos()
		actualLines := int(cursorPos.Y) + 1

		const minPanelHeight = 3
		const maxPanelCap = 15

		panelHeight := min(actualLines, maxPanelCap)
		panelHeight = max(panelHeight, min(actualLines, minPanelHeight))

		prunedCount := actualLines - panelHeight
		if prunedCount > 0 {
			// Pruned indicator takes 1 row, content is reduced by 1
			h += 1 + panelHeight - 1 + 1 // pruned indicator + content + spacing
			// Simplifies to: h += panelHeight + 1
		} else {
			h += panelHeight + 1 // content + spacing
		}
	}
	return h
}
