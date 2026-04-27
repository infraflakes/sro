package tui

import (
	"github.com/gdamore/tcell/v3/color"
)

// Color palette matching the JSX C object from the plan
var (
	// Background colors
	Bg  = color.NewRGBColor(0x08, 0x0a, 0x0f) // #080a0f
	Bg1 = color.NewRGBColor(0x0d, 0x10, 0x17) // #0d1017
	Bg2 = color.NewRGBColor(0x11, 0x15, 0x20) // #111520
	Bg3 = color.NewRGBColor(0x16, 0x1b, 0x28) // #161b28

	// Status colors
	Ok      = color.NewRGBColor(0x4e, 0xc9, 0xa0) // #4ec9a0
	Running = color.NewRGBColor(0x5b, 0x9c, 0xf6) // #5b9cf6
	Failed  = color.NewRGBColor(0xe0, 0x5c, 0x6a) // #e05c6a
	Pending = color.NewRGBColor(0x4a, 0x58, 0x78) // #4a5878

	// Type colors
	Seq  = color.NewRGBColor(0xc7, 0x92, 0xea) // #c792ea
	Par  = color.NewRGBColor(0x5b, 0x9c, 0xf6) // #5b9cf6
	Sync = color.NewRGBColor(0x4e, 0xc9, 0xa0) // #4ec9a0

	// Text colors
	Muted      = color.NewRGBColor(0x4a, 0x58, 0x78) // #4a5878
	Text       = color.NewRGBColor(0xb8, 0xc4, 0xe8) // #b8c4e8
	TextBright = color.NewRGBColor(0xd8, 0xe2, 0xf8) // #d8e2f8
	Dim        = color.NewRGBColor(0x2a, 0x35, 0x48) // #2a3548

	// Annotation colors
	LogC  = color.NewRGBColor(0xff, 0xcb, 0x6b) // #ffcb6b
	ExecC = color.NewRGBColor(0x5b, 0x9c, 0xf6) // #5b9cf6
	CdC   = color.NewRGBColor(0xff, 0xcb, 0x6b) // #ffcb6b
	EnvC  = color.NewRGBColor(0xc7, 0x92, 0xea) // #c792ea
)
