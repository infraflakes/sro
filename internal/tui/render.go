package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

func Render(screen tcell.Screen, model *Model, spinnerIdx int) {
	w, h := screen.Size()
	screen.Fill(' ', tcell.StyleDefault.Background(Bg))

	y := 0

	// Header
	renderHeader(screen, model, w, &y)
	y++

	// Type note
	renderTypeNote(screen, model, w, &y)
	y++

	// Task rows
	for i := range model.Tasks {
		renderTaskRow(screen, model, i, w, &y, spinnerIdx)
		if model.Tasks[i].Expanded {
			renderExpandedPanel(screen, &model.Tasks[i], w, &y)
		}
	}

	// Footer
	renderFooter(screen, model, w, h)
}

func renderHeader(screen tcell.Screen, model *Model, w int, y *int) {
	var badgeColor tcell.Color
	switch model.Type {
	case "seq":
		badgeColor = Seq
	case "par":
		badgeColor = Par
	case "sync":
		badgeColor = Sync
	default:
		badgeColor = Text
	}

	badgeText := fmt.Sprintf(" %s ", model.Type)
	for i, r := range badgeText {
		style := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(badgeColor)
		screen.SetContent(i, *y, r, nil, style)
	}

	nameText := fmt.Sprintf(" %s ", model.Name)
	for i, r := range nameText {
		style := tcell.StyleDefault.Foreground(TextBright).Background(Bg1)
		screen.SetContent(len(badgeText)+i, *y, r, nil, style)
	}

	countText := fmt.Sprintf(" %d tasks ", len(model.Tasks))
	offset := len(badgeText) + len(nameText)
	for i, r := range countText {
		style := tcell.StyleDefault.Foreground(Muted).Background(Bg1)
		screen.SetContent(offset+i, *y, r, nil, style)
	}

	// Status with spinner
	var statusText string
	var statusColor tcell.Color
	switch model.Status {
	case "running":
		statusText = " running"
		statusColor = Running
	case "ok":
		statusText = " ok"
		statusColor = Ok
	case "failed":
		statusText = " failed"
		statusColor = Failed
	default:
		statusText = " unknown"
		statusColor = Muted
	}

	statusOffset := w - len(statusText)
	for i, r := range statusText {
		style := tcell.StyleDefault.Foreground(statusColor).Background(Bg1)
		screen.SetContent(statusOffset+i, *y, r, nil, style)
	}
}

func renderTypeNote(screen tcell.Screen, model *Model, w int, y *int) {
	var note string
	switch model.Type {
	case "seq":
		note = "seq — sequential · fail-fast · one active at a time"
	case "par":
		note = "par — parallel · all tasks run simultaneously"
	case "sync":
		note = "sync — clone/sync project repositories"
	default:
		note = ""
	}

	for i, r := range note {
		style := tcell.StyleDefault.Foreground(Muted).Background(Bg)
		screen.SetContent(i, *y, r, nil, style)
	}
}

func renderTaskRow(screen tcell.Screen, model *Model, taskIdx int, w int, y *int, spinnerIdx int) {
	task := &model.Tasks[taskIdx]

	// Status icon
	var icon rune
	var iconColor color.Color
	switch task.Status {
	case "ok":
		icon = '✓'
		iconColor = Ok
	case "failed":
		icon = '✗'
		iconColor = Failed
	case "running":
		icon = SpinnerFrames[spinnerIdx]
		iconColor = Running
	case "pending":
		icon = '·'
		iconColor = Pending
	default:
		icon = '?'
		iconColor = Muted
	}

	screen.SetContent(2, *y, icon, nil, tcell.StyleDefault.Foreground(iconColor).Background(Bg))

	// Label
	for i, r := range task.Label {
		style := tcell.StyleDefault.Foreground(Text).Background(Bg)
		screen.SetContent(4+i, *y, r, nil, style)
	}

	// Expand arrow
	arrow := '▶'
	if task.Expanded {
		arrow = '▼'
	}
	screen.SetContent(w-2, *y, arrow, nil, tcell.StyleDefault.Foreground(Muted).Background(Bg))

	// Vertical connector bar for seq
	if model.Type == "seq" && taskIdx > 0 {
		connectorColor := Ok
		if task.Status == "failed" {
			connectorColor = Failed
		} else if task.Status == "running" {
			connectorColor = Running
		}
		screen.SetContent(2, *y-1, '│', nil, tcell.StyleDefault.Foreground(connectorColor).Background(Bg))
	}

	*y++
}

func renderExpandedPanel(screen tcell.Screen, task *Task, w int, y *int) {
	panelHeight := 10 // Fixed height for now
	panelWidth := w - 4

	// Top border matching task status
	var borderColor color.Color
	switch task.Status {
	case "ok":
		borderColor = Ok
	case "failed":
		borderColor = Failed
	case "running":
		borderColor = Running
	default:
		borderColor = Muted
	}

	for x := 2; x < w-2; x++ {
		screen.SetContent(x, *y, '─', nil, tcell.StyleDefault.Foreground(borderColor).Background(Bg))
	}
	*y++

	// Render output from buffer
	if task.Output != nil {
		lines := strings.Split(task.Output.String(), "\n")
		// Show last N lines
		startLine := max(0, len(lines)-panelHeight)
		for i := startLine; i < len(lines) && i < startLine+panelHeight; i++ {
			line := lines[i]
			for j, r := range line {
				if j < panelWidth {
					screen.SetContent(2+j, *y, r, nil, tcell.StyleDefault.Foreground(Text).Background(Bg))
				}
			}
			*y++
		}
	} else {
		*y += panelHeight
	}

	// Bottom border with spinner if running
	if task.Status == "running" {
		spinnerText := " ⠋ running"
		for i, r := range spinnerText {
			style := tcell.StyleDefault.Foreground(Running).Background(Bg)
			screen.SetContent(2+i, *y, r, nil, style)
		}
	}
	*y++
}

func renderFooter(screen tcell.Screen, model *Model, w, h int) {
	y := h - 1

	var okCount, runningCount, pendingCount, failedCount int
	for _, task := range model.Tasks {
		switch task.Status {
		case "ok":
			okCount++
		case "running":
			runningCount++
		case "pending":
			pendingCount++
		case "failed":
			failedCount++
		}
	}

	footerText := fmt.Sprintf(" %d tasks | %d ok | %d running | %d pending | %d failed ",
		len(model.Tasks), okCount, runningCount, pendingCount, failedCount)

	for i, r := range footerText {
		style := tcell.StyleDefault.Foreground(Muted).Background(Bg1)
		screen.SetContent(i, y, r, nil, style)
	}
}
