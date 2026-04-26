package tui

import (
	"bytes"
	"io"
)

type Model struct {
	Type     string // "seq", "par", "sync"
	Name     string
	Status   string // "running", "ok", "failed"
	Tasks    []Task
	Selected int // keyboard cursor position
}

type Task struct {
	Label    string
	Status   string // "ok", "running", "failed", "pending"
	Expanded bool
	Output   *bytes.Buffer // captured output for this task
	Writer   io.Writer     // writer for piping output
}
