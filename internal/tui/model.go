package tui

import (
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
	Label    string
	Status   string // "ok", "running", "failed", "pending"
	Expanded bool
	VTerm    vt.MockTerm // virtual terminal for this task
}
