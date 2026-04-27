package tui

import (
	"io"

	"github.com/gdamore/tcell/v3/vt"
)

type Model struct {
	Type         string // "seq", "par", "sync"
	Name         string
	Status       string // "running", "ok", "failed"
	Tasks        []Task
	Selected     int // keyboard cursor position
	ScrollOffset int // viewport scroll offset
}

type Task struct {
	Label      string
	Status     string // "ok", "running", "failed", "pending"
	Expanded   bool
	VTerm      vt.MockTerm // virtual terminal for this task
	TotalLines int         // total lines output (for accurate pruning calculation)
}

// lineCountingWriter wraps an io.Writer to count newline characters
type lineCountingWriter struct {
	writer     io.Writer
	totalLines *int
}

func (w *lineCountingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err != nil {
		return n, err
	}
	// Count newlines in the written data
	for _, b := range p {
		if b == '\n' {
			*w.totalLines++
		}
	}
	return n, nil
}

// NewLineCountingWriter creates a writer that counts newlines written to it
func NewLineCountingWriter(writer io.Writer, totalLines *int) io.Writer {
	return &lineCountingWriter{
		writer:     writer,
		totalLines: totalLines,
	}
}
