package tui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v3"
)

func Run(model *Model) error {
	return RunWithContext(context.Background(), model)
}

func RunWithContext(ctx context.Context, model *Model) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault)

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	spinnerIdx := 0
	quit := false

	for !quit {
		select {
		case <-ctx.Done():
			quit = true
		case ev := <-screen.EventQ():
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					quit = true
				case tcell.KeyUp, tcell.KeyCtrlP:
					model.Mu.Lock()
					if model.Selected > 0 {
						model.Selected--
					}
					model.Mu.Unlock()
				case tcell.KeyDown, tcell.KeyCtrlN:
					model.Mu.Lock()
					if model.Selected < len(model.Tasks)-1 {
						model.Selected++
					}
					model.Mu.Unlock()
				case tcell.KeyEnter:
					model.Mu.Lock()
					if model.Selected >= 0 && model.Selected < len(model.Tasks) {
						// Don't allow expansion of pending tasks in seq mode
						if model.Type != "seq" || model.Tasks[model.Selected].Status != "pending" {
							model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
						}
					}
					model.Mu.Unlock()
				case tcell.KeyRune:
					if ev.Str() == " " {
						model.Mu.Lock()
						if model.Selected >= 0 && model.Selected < len(model.Tasks) {
							// Don't allow expansion of pending tasks in seq mode
							if model.Type != "seq" || model.Tasks[model.Selected].Status != "pending" {
								model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
							}
						}
						model.Mu.Unlock()
					} else if ev.Str() == "q" {
						quit = true
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		case <-ticker.C:
			spinnerIdx = (spinnerIdx + 1) % len(SpinnerFrames)
		}

		// Adjust scroll offset to keep selected task visible
		_, h := screen.Size()
		headerHeight := 4
		footerHeight := 4

		// Calculate the rendered height of each task (1 row collapsed, 1 + panelHeight expanded)
		model.Mu.Lock()
		taskHeights := make([]int, len(model.Tasks))
		for i, task := range model.Tasks {
			taskHeights[i] = 1 // collapsed row
			if task.Expanded && task.VTerm != nil {
				cursorPos := task.VTerm.Pos()
				actualLines := int(cursorPos.Y) + 1
				panelHeight := min(actualLines, 15)
				panelHeight = max(panelHeight, min(actualLines, 3))
				if actualLines > panelHeight {
					panelHeight++ // pruned indicator row
				}
				taskHeights[i] += panelHeight + 1 // +1 for spacing after panel
			}
		}
		visibleHeight := h - headerHeight - footerHeight

		// Calculate the Y position of the selected task relative to ScrollOffset
		yBefore := 0
		for i := model.ScrollOffset; i < model.Selected; i++ {
			if i < len(taskHeights) {
				yBefore += taskHeights[i]
			}
		}

		// If selected task is above the viewport, scroll up
		if model.Selected < model.ScrollOffset {
			model.ScrollOffset = model.Selected
		}

		// If selected task (including its expanded panel) is below the viewport, scroll down
		selectedHeight := 0
		if model.Selected < len(taskHeights) {
			selectedHeight = taskHeights[model.Selected]
		}
		if yBefore+selectedHeight > visibleHeight {
			// Scroll so the selected task's bottom edge is visible
			// Walk backwards from Selected to find the right ScrollOffset
			remaining := visibleHeight
			model.ScrollOffset = model.Selected
			for model.ScrollOffset > 0 {
				remaining -= taskHeights[model.ScrollOffset]
				if remaining <= 0 {
					break
				}
				model.ScrollOffset--
			}
			// Ensure we don't scroll past the selected task
			if model.ScrollOffset > model.Selected {
				model.ScrollOffset = model.Selected
			}
		}
		model.Mu.Unlock()

		// Update spinner in model status
		// Spinner will be rendered in render()

		Render(screen, model, spinnerIdx)
		screen.Show()
	}

	// Stop all vterms on exit
	model.Mu.Lock()
	for i := range model.Tasks {
		if model.Tasks[i].VTerm != nil {
			_ = model.Tasks[i].VTerm.Stop()
		}
	}
	model.Mu.Unlock()

	return nil
}
