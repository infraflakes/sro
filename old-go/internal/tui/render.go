package tui

import (
	"fmt"
	"sync/atomic"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/mattn/go-runewidth"
)

var SpinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

func Render(screen tcell.Screen, model *Model, spinnerIdx int) {
	w, h := screen.Size()
	screen.Fill(' ', tcell.StyleDefault)

	// Fixed footer height including its separator and spacing
	footerHeight := 4
	footerY := h - footerHeight // first row reserved for footer

	y := 0

	// Snapshot all model fields under lock for safe concurrent access
	model.Mu.Lock()
	modelType := model.Type
	modelName := model.Name
	selected := model.Selected
	tasksCopy := make([]Task, len(model.Tasks))
	copy(tasksCopy, model.Tasks)
	scrollOffset := model.ScrollOffset
	model.Mu.Unlock()

	// Header (includes its own separator and spacing)
	renderHeader(screen, modelType, modelName, len(tasksCopy), w, &y)

	// Task rows — render until we run out of screen space
	for i := scrollOffset; i < len(tasksCopy); i++ {
		if y >= footerY {
			break
		}
		renderTaskRow(screen, tasksCopy, i, w, &y, spinnerIdx, selected)
		if y >= footerY {
			break // task row itself hit the boundary
		}
		if tasksCopy[i].Expanded {
			// Hold lock during VTerm access to prevent race with concurrent writes
			model.Mu.Lock()
			renderExpandedPanel(screen, &tasksCopy[i], w, &y, footerY)
			model.Mu.Unlock()
		}
	}

	// Footer (includes its own separator and spacing)
	renderFooter(screen, tasksCopy, w, h, spinnerIdx)
}

func renderHeader(screen tcell.Screen, modelType string, modelName string, taskCount int, w int, y *int) {
	var badgeColor tcell.Color
	switch modelType {
	case "seq":
		badgeColor = Seq
	case "par":
		badgeColor = Par
	case "sync":
		badgeColor = Sync
	default:
		badgeColor = Text
	}

	badgeText := fmt.Sprintf(" %s ", modelType)
	for i, r := range badgeText {
		style := tcell.StyleDefault.Foreground(color.Black).Background(badgeColor)
		screen.SetContent(i, *y, r, nil, style)
	}

	nameText := fmt.Sprintf(" %s ", modelName)
	for i, r := range nameText {
		style := tcell.StyleDefault.Foreground(TextBright)
		screen.SetContent(len(badgeText)+i, *y, r, nil, style)
	}

	countText := fmt.Sprintf(" %d tasks ", taskCount)
	offset := w - len(countText)
	for i, r := range countText {
		style := tcell.StyleDefault.Foreground(Muted)
		screen.SetContent(offset+i, *y, r, nil, style)
	}

	*y++

	// Horizontal separator line
	for x := range w {
		screen.SetContent(x, *y, '─', nil, tcell.StyleDefault.Foreground(Muted))
	}
	*y++

	// Vertical spacing (2 lines)
	for range 2 {
		*y++
	}
}

func renderTaskRow(screen tcell.Screen, tasks []Task, taskIdx int, w int, y *int, spinnerIdx int, selected int) {
	task := &tasks[taskIdx]
	isSelected := taskIdx == selected

	// Clear the row with default background
	for x := range w {
		screen.SetContent(x, *y, ' ', nil, tcell.StyleDefault)
	}

	// Selection indicator
	if isSelected {
		screen.SetContent(0, *y, '▸', nil, tcell.StyleDefault.Foreground(TextBright))
	}

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

	screen.SetContent(2, *y, icon, nil, tcell.StyleDefault.Foreground(iconColor))

	// Label
	labelColor := Text
	if isSelected {
		labelColor = TextBright
	}
	col := 4
	style := tcell.StyleDefault.Foreground(labelColor)
	if isSelected {
		style = style.Bold(true)
	}
	for _, r := range task.Label {
		screen.SetContent(col, *y, r, nil, style)
		col += runewidth.RuneWidth(r)
	}

	// Expand arrow (hide for pending tasks in seq mode)
	// Note: model.Type is not available here, but this check is handled by the executor
	if task.Status != "pending" {
		arrow := '▶'
		if task.Expanded {
			arrow = '▼'
		}
		screen.SetContent(w-2, *y, arrow, nil, tcell.StyleDefault.Foreground(Muted))
	}

	*y++
}

func renderExpandedPanel(screen tcell.Screen, task *Task, w int, y *int, maxY int) {
	panelWidth := w - 4

	const minPanelHeight = 3 // never prune below this, even if terminal is tiny
	const maxPanelCap = 15   // max lines before pruning kicks in

	// Blit cells from vterm
	if task.VTerm != nil {
		be := task.VTerm.Backend()
		vtSize := be.GetSize()

		// Available rows between current y and the footer boundary
		// Reserve 1 row for spacing after panel
		availableRows := maxY - *y - 1
		if availableRows < 1 {
			return // no room at all
		}

		cursorPos := task.VTerm.Pos()
		actualLines := int(cursorPos.Y) + 1

		// Panel height is based on actual content, capped at maxPanelCap
		panelHeight := min(actualLines, maxPanelCap)
		panelHeight = max(panelHeight, min(actualLines, minPanelHeight))
		panelHeight = min(panelHeight, availableRows)

		// For pruning display, use TotalLines if available to show true count
		prunedCount := actualLines - panelHeight
		totalLines := atomic.LoadInt64(&task.TotalLines)
		if totalLines > 0 {
			prunedCount = int(totalLines) - panelHeight
		}
		startRow := max(actualLines-panelHeight, 0)

		// Pruned indicator (takes 1 row from the panel budget)
		if prunedCount > 0 {
			if *y < maxY {
				prunedText := fmt.Sprintf(" ↑ %d lines hidden ", prunedCount)
				for i, r := range prunedText {
					screen.SetContent(2+i, *y, r, nil, tcell.StyleDefault.Foreground(Dim))
				}
			}
			*y++
			panelHeight = max(1, panelHeight-1)
			startRow = max(actualLines-panelHeight, 0)
		}

		// Blit cells — hard stop at maxY
		for row := 0; row < panelHeight; row++ {
			if *y >= maxY {
				break
			}
			vtRow := vt.Row(startRow + row)
			if vtRow >= vtSize.Y {
				break
			}
			for col := vt.Col(0); col < vtSize.X && int(col) < panelWidth; col++ {
				cell := be.GetCell(vt.Coord{X: col, Y: vtRow})
				if cell.C != "" {
					style := vtStyleToTcellStyle(cell.S)
					r := []rune(cell.C)
					if len(r) > 0 {
						screen.SetContent(2+int(col), *y, r[0], r[1:], style)
					}
				}
			}
			*y++
		}
	} else {
		if *y < maxY {
			*y++
		}
	}

	// Spacing after panel — only if there's room
	if *y < maxY {
		*y++
	}
}

func renderFooter(screen tcell.Screen, tasks []Task, w, h int, spinnerIdx int) {
	footerStart := h - 4 // footer zone is 4 rows tall

	// Clear all footer rows
	for row := footerStart; row < h; row++ {
		for x := range w {
			screen.SetContent(x, row, ' ', nil, tcell.StyleDefault)
		}
	}

	// Start from bottom and work up
	y := h - 1

	var okCount, runningCount, pendingCount, failedCount int
	for _, task := range tasks {
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

	x := 1
	// Only render non-zero counts, with colored icons
	if okCount > 0 {
		x += drawText(screen, x, y, fmt.Sprintf("✓ %d ok", okCount), Ok, color.Default)
		x += 2 // gap
	}
	if runningCount > 0 {
		spinnerChar := SpinnerFrames[spinnerIdx]
		x += drawText(screen, x, y, fmt.Sprintf("%c %d running", spinnerChar, runningCount), Running, color.Default)
		x += 2
	}
	if pendingCount > 0 {
		x += drawText(screen, x, y, fmt.Sprintf("· %d pending", pendingCount), Pending, color.Default)
		x += 2
	}
	if failedCount > 0 {
		drawText(screen, x, y, fmt.Sprintf("✗ %d failed", failedCount), Failed, color.Default)
	}

	// Move up for vertical spacing (2 lines)
	y -= 2

	// Horizontal separator line
	for x := range w {
		screen.SetContent(x, y, '─', nil, tcell.StyleDefault.Foreground(Muted))
	}
}

// Helper to draw colored text, returns number of cells written
func drawText(screen tcell.Screen, x, y int, text string, fg, bg color.Color) int {
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	i := 0
	for _, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
		i++
	}
	return i
}

func vtStyleToTcellStyle(vs vt.Style) tcell.Style {
	ts := tcell.StyleDefault

	fg := vs.Fg()
	// color.Silver (XTerm7) is the vterm emulator's default foreground — treat as terminal default
	if fg != color.Default && fg != color.Silver {
		ts = ts.Foreground(tcell.Color(fg))
	}
	bg := vs.Bg()
	// color.Black (XTerm0) is the vterm emulator's default background — treat as terminal default
	if bg != color.Default && bg != color.Black {
		ts = ts.Background(tcell.Color(bg))
	}

	attrs := vs.Attr()
	if attrs&vt.Bold != 0 {
		ts = ts.Bold(true)
	}
	if attrs&vt.Italic != 0 {
		ts = ts.Italic(true)
	}
	if attrs&vt.Underline != 0 {
		ts = ts.Underline(true)
	}
	if attrs&vt.Dim != 0 {
		ts = ts.Dim(true)
	}
	if attrs&vt.StrikeThrough != 0 {
		ts = ts.StrikeThrough(true)
	}
	if attrs&vt.Reverse != 0 {
		ts = ts.Reverse(true)
	}
	return ts
}
