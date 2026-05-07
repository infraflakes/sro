package tui

import (
	"testing"
)

func TestSpinnerFrames(t *testing.T) {
	// Verify spinner frames are defined
	if len(SpinnerFrames) == 0 {
		t.Fatal("SpinnerFrames is empty")
	}
}

func TestModelDefaults(t *testing.T) {
	model := &Model{
		Type:         "seq",
		Name:         "test",
		Status:       "running",
		Selected:     0,
		ScrollOffset: 0,
		Tasks: []Task{
			{Label: "task1", Status: "ok", Expanded: false},
		},
	}

	if model.Type != "seq" {
		t.Errorf("expected Type=seq, got %s", model.Type)
	}
	if model.Selected != 0 {
		t.Errorf("expected Selected=0, got %d", model.Selected)
	}
	if len(model.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(model.Tasks))
	}
}
