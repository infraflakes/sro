package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/gdamore/tcell/v3/vt"
)

func Render(screen tcell.Screen, model *Model, spinnerIdx int) {
	w, h := screen.Size()
	screen.Fill(' ', tcell.StyleDefault)

	// Calculate header height
	headerHeight := 4 // header + 2 spacing lines
	footerHeight := 1 // footer only
	visibleHeight := h - headerHeight - footerHeight

	// Calculate which tasks are visible based on ScrollOffset
	startTask := model.ScrollOffset
	endTask := min(startTask+visibleHeight, len(model.Tasks))

	y := 0

	// Header
	renderHeader(screen, model, w, &y)
	y++

	// Add vertical spacing between header and tasks
	for range 2 {
		y++
	}

	// Task rows (only visible ones)
	for i := startTask; i < endTask; i++ {
		renderTaskRow(screen, model, i, w, &y, spinnerIdx)
		if model.Tasks[i].Expanded {
			renderExpandedPanel(screen, &model.Tasks[i], w, &y)
		}
	}

	// Footer
	renderFooter(screen, model, w, h, spinnerIdx)
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
		style := tcell.StyleDefault.Foreground(TextBright)
		screen.SetContent(len(badgeText)+i, *y, r, nil, style)
	}

	countText := fmt.Sprintf(" %d tasks ", len(model.Tasks))
	offset := w - len(countText)
	for i, r := range countText {
		style := tcell.StyleDefault.Foreground(Muted)
		screen.SetContent(offset+i, *y, r, nil, style)
	}

	*y++
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
		style := tcell.StyleDefault.Foreground(Muted)
		screen.SetContent(i, *y, r, nil, style)
	}
}

func renderTaskRow(screen tcell.Screen, model *Model, taskIdx int, w int, y *int, spinnerIdx int) {
	task := &model.Tasks[taskIdx]
	isSelected := taskIdx == model.Selected

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
	for i, r := range task.Label {
		style := tcell.StyleDefault.Foreground(labelColor)
		if isSelected {
			style = style.Bold(true)
		}
		screen.SetContent(4+i, *y, r, nil, style)
	}

	// Expand arrow (hide for pending tasks in seq mode)
	if !(model.Type == "seq" && task.Status == "pending") {
		arrow := '▶'
		if task.Expanded {
			arrow = '▼'
		}
		screen.SetContent(w-2, *y, arrow, nil, tcell.StyleDefault.Foreground(Muted))
	}

	*y++
}

func renderExpandedPanel(screen tcell.Screen, task *Task, w int, y *int) {
	_, termH := screen.Size()
	availableHeight := termH - *y - 3 // leave room for footer and remaining tasks
	panelWidth := w - 4

	// Blit cells from vterm
	if task.VTerm != nil {
		cursorPos := task.VTerm.Pos()
		actualLines := int(cursorPos.Y) + 1 // cursor Y is 0-indexed

		// Cap at a max threshold, but don't exceed available space
		maxPanelHeight := min(15, max(1, availableHeight))
		panelHeight := min(actualLines, maxPanelHeight)

		// Number of pruned (hidden) lines
		prunedCount := actualLines - panelHeight
		startRow := max(actualLines-panelHeight, 0)

		// Pruned indicator (only if lines were hidden)
		if prunedCount > 0 {
			prunedText := fmt.Sprintf(" ↑ %d lines hidden ", prunedCount)
			for i, r := range prunedText {
				screen.SetContent(2+i, *y, r, nil, tcell.StyleDefault.Foreground(Dim))
			}
			*y++
			// Reduce panelHeight by 1 to account for the pruned indicator row
			panelHeight = max(1, panelHeight-1)
			startRow = max(actualLines-panelHeight, 0)
		}

		be := task.VTerm.Backend()
		vtSize := be.GetSize()
		for row := 0; row < panelHeight && vt.Row(startRow+row) < vtSize.Y; row++ {
			for col := vt.Col(0); col < vtSize.X && int(col) < panelWidth; col++ {
				cell := be.GetCell(vt.Coord{X: col, Y: vt.Row(startRow + row)})
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
		// No vterm — just skip 1 line
		*y++
	}

	*y++ // spacing after panel
}

func renderFooter(screen tcell.Screen, model *Model, w, h int, spinnerIdx int) {
	y := h - 3

	// Clear footer row
	for x := range w {
		screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
	}

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
		x += drawText(screen, x, y, fmt.Sprintf("✗ %d failed", failedCount), Failed, color.Default)
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
