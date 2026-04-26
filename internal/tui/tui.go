package tui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v3"
)

func Run(model *Model) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault.Background(Bg))

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	spinnerIdx := 0
	quit := false

	for !quit {
		select {
		case ev := <-screen.EventQ():
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					quit = true
				case tcell.KeyUp, tcell.KeyCtrlN:
					if model.Selected > 0 {
						model.Selected--
					}
				case tcell.KeyDown, tcell.KeyCtrlP:
					if model.Selected < len(model.Tasks)-1 {
						model.Selected++
					}
				case tcell.KeyEnter:
					if model.Selected >= 0 && model.Selected < len(model.Tasks) {
						model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
					}
				case tcell.KeyRune:
					if ev.Str() == " " {
						if model.Selected >= 0 && model.Selected < len(model.Tasks) {
							model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
						}
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

		// Update spinner in model status
		if model.Status == "running" {
			// Update spinner for running tasks
			for i := range model.Tasks {
				if model.Tasks[i].Status == "running" {
					// Spinner will be rendered in render()
				}
			}
		}

		Render(screen, model, spinnerIdx)
		screen.Show()

		// Check if all tasks are done
		allDone := true
		for _, task := range model.Tasks {
			if task.Status == "running" || task.Status == "pending" {
				allDone = false
				break
			}
		}
		if allDone && model.Status == "running" {
			// Determine final status
			hasFailed := false
			for _, task := range model.Tasks {
				if task.Status == "failed" {
					hasFailed = true
					break
				}
			}
			if hasFailed {
				model.Status = "failed"
			} else {
				model.Status = "ok"
			}
		}
	}

	return nil
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

	screen.SetStyle(tcell.StyleDefault.Background(Bg))

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
				case tcell.KeyUp, tcell.KeyCtrlN:
					if model.Selected > 0 {
						model.Selected--
					}
				case tcell.KeyDown, tcell.KeyCtrlP:
					if model.Selected < len(model.Tasks)-1 {
						model.Selected++
					}
				case tcell.KeyEnter:
					if model.Selected >= 0 && model.Selected < len(model.Tasks) {
						model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
					}
				case tcell.KeyRune:
					if ev.Str() == " " {
						if model.Selected >= 0 && model.Selected < len(model.Tasks) {
							model.Tasks[model.Selected].Expanded = !model.Tasks[model.Selected].Expanded
						}
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

		// Update spinner in model status
		if model.Status == "running" {
			// Update spinner for running tasks
			for i := range model.Tasks {
				if model.Tasks[i].Status == "running" {
					// Spinner will be rendered in render()
				}
			}
		}

		Render(screen, model, spinnerIdx)
		screen.Show()

		// Check if all tasks are done
		allDone := true
		for _, task := range model.Tasks {
			if task.Status == "running" || task.Status == "pending" {
				allDone = false
				break
			}
		}
		if allDone && model.Status == "running" {
			// Determine final status
			hasFailed := false
			for _, task := range model.Tasks {
				if task.Status == "failed" {
					hasFailed = true
					break
				}
			}
			if hasFailed {
				model.Status = "failed"
			} else {
				model.Status = "ok"
			}
		}
	}

	return nil
}
