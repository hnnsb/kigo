package editor

import (
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-runewidth"
)

// ScreenRenderer is responsible for drawing editor state to the terminal.
type ScreenRenderer struct{}

func NewScreenRenderer() *ScreenRenderer {
	return &ScreenRenderer{}
}

func runeDisplayWidth(r rune) int {
	width := runewidth.RuneWidth(r)
	if width < 0 {
		return 1
	}
	return width
}

func renderStartIndex(render []rune, offset int) int {
	if offset <= 0 {
		return 0
	}

	visibleWidth := 0
	for index, r := range render {
		visibleWidth += runeDisplayWidth(r)
		if visibleWidth > offset {
			return index
		}
	}

	return len(render)
}

func appendDisplayLine(abuf *appendBuffer, line DisplayLine, startOffset int, maxWidth int, pad bool) {
	if maxWidth <= 0 {
		return
	}

	start := renderStartIndex(line.render, startOffset)
	visibleWidth := 0
	currentColor := -1
	currentStyle := 0

	for i := start; i < len(line.render) && visibleWidth < maxWidth; i++ {
		c := line.render[i]
		h := line.hl[i]
		charWidth := runeDisplayWidth(c)
		if charWidth > 0 && visibleWidth+charWidth > maxWidth {
			break
		}

		if h == HL_NORMAL {
			if currentColor != -1 {
				abuf.append(fmt.Appendf(nil, "\x1b[%dm", ANSI_COLOR_DEFAULT))
				currentColor = -1
			}
			if currentStyle != 0 {
				resetCode := getStyleResetCode(currentStyle)
				if resetCode != 0 {
					abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
				}
				currentStyle = 0
			}
			abuf.append([]byte(string(c)))
		} else {
			color, style := syntaxToGraphics(h)
			if currentStyle != style {
				if currentStyle != 0 {
					resetCode := getStyleResetCode(currentStyle)
					if resetCode != 0 {
						abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
					}
				}
				if style != 0 {
					abuf.append(fmt.Appendf(nil, "\x1b[%dm", style))
				}
				currentStyle = style
			}
			if color != currentColor {
				currentColor = color
				abuf.append(fmt.Appendf(nil, "\x1b[%dm", color))
			}
			abuf.append([]byte(string(c)))
		}

		if charWidth > 0 {
			visibleWidth += charWidth
		}
	}

	abuf.append(fmt.Appendf(nil, "\x1b[%dm", ANSI_COLOR_DEFAULT))
	if currentStyle != 0 {
		resetCode := getStyleResetCode(currentStyle)
		if resetCode != 0 {
			abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
		}
	}

	if pad {
		for visibleWidth < maxWidth {
			abuf.append([]byte(" "))
			visibleWidth++
		}
	}
}

func appendPreviewLine(abuf *appendBuffer, text string, maxWidth int) {
	if maxWidth <= 0 {
		return
	}

	visibleWidth := 0
	for _, r := range text {
		charWidth := runeDisplayWidth(r)
		if charWidth > 0 && visibleWidth+charWidth > maxWidth {
			break
		}
		abuf.append([]byte(string(r)))
		if charWidth > 0 {
			visibleWidth += charWidth
		}
	}
}

func (r *ScreenRenderer) drawEditorRows(e *Editor, abuf *appendBuffer) {
	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if filerow >= e.totalRows {
			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + KIGO_VERSION
				welcomelen := min(len(welcome), e.screenCols)
				padding := (e.screenCols - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte("~"))
					padding--
				}
				for range padding {
					abuf.append([]byte(" "))
				}
				abuf.append([]byte(welcome[:welcomelen]))
			} else {
				abuf.append([]byte("~"))
			}
		} else {
			appendDisplayLine(abuf, e.row[filerow], e.colOffset, e.screenCols, true)
		}
		abuf.append([]byte(CLEAR_LINE))
		abuf.append([]byte("\r\n"))
	}
}

func (r *ScreenRenderer) DrawRows(e *Editor, abuf *appendBuffer) {
	if e.mode == EXPLORER_MODE && e.activeModal != nil {
		if splitViewModal, ok := e.activeModal.(SplitViewModal); ok && splitViewModal.ShouldShowSplitView(e.screenCols) {
			r.drawSplitViewRows(e, abuf, splitViewModal)
			return
		}
	}
	r.drawEditorRows(e, abuf)
}

func (r *ScreenRenderer) drawSplitViewRows(e *Editor, abuf *appendBuffer, splitModal SplitViewModal) {
	leftWidth := e.screenCols / 2
	rightWidth := e.screenCols - leftWidth - 1

	if rightWidth < MIN_SPLIT_PANE_WIDTH {
		r.drawEditorRows(e, abuf)
		return
	}

	_, rightPreview := splitModal.GetSplitViewContent(e, rightWidth, e.screenRows)

	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if filerow >= e.totalRows {
			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + KIGO_VERSION
				welcomelen := min(len(welcome), leftWidth)
				padding := (leftWidth - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte("~"))
					padding--
				}
				for range padding {
					abuf.append([]byte(" "))
				}
				abuf.append([]byte(welcome[:welcomelen]))
				for i := welcomelen + padding + 1; i < leftWidth; i++ {
					abuf.append([]byte(" "))
				}
			} else {
				abuf.append([]byte("~"))
				for i := 1; i < leftWidth; i++ {
					abuf.append([]byte(" "))
				}
			}
		} else {
			appendDisplayLine(abuf, e.row[filerow], e.colOffset, leftWidth, true)
		}

		abuf.append([]byte("|"))
		if y < len(rightPreview) {
			appendPreviewLine(abuf, rightPreview[y], rightWidth)
		} else {
			for range rightWidth {
				abuf.append([]byte(" "))
			}
		}
		abuf.append([]byte(CLEAR_LINE))
		abuf.append([]byte("\r\n"))
	}
}

func (r *ScreenRenderer) DrawStatusBar(e *Editor, abuf *appendBuffer) {
	abuf.append([]byte(COLORS_INVERT))

	var status string
	var rstatus string
	filename := "[No Name]"
	if e.filename != "" {
		filename = e.filename
		if len(filename) > 20 {
			filename = filename[:20]
		}
	}
	dirtyFlag := ""
	if e.dirty > 0 {
		dirtyFlag = "(modified)"
	}
	switch e.mode {
	case EXPLORER_MODE:
		status = fmt.Sprintf("Explorer - %s %s", filename, dirtyFlag)
	default:
		status = fmt.Sprintf("%.20s - %d lines %s %d", filename, e.totalRows, dirtyFlag, e.dirty)
	}
	statusLen := min(len(status), e.screenCols)

	filetype := "no ft"
	if e.syntax != nil {
		filetype = e.syntax.filetype
	}
	rstatus = fmt.Sprintf("%s | %d/%d", filetype, e.cy+1, e.totalRows)
	rstatusLen := len(rstatus)
	abuf.append([]byte(status[:statusLen]))

	for statusLen < e.screenCols {
		if e.screenCols-statusLen == rstatusLen {
			abuf.append([]byte(rstatus))
			break
		}
		abuf.append([]byte(" "))
		statusLen++
	}

	abuf.append([]byte(COLORS_RESET))
	abuf.append([]byte("\r\n"))
}

func (r *ScreenRenderer) DrawMessageBar(e *Editor, abuf *appendBuffer) {
	abuf.append([]byte(CLEAR_LINE))
	messageLen := min(len(e.statusMessage), e.screenCols)
	if time.Since(e.statusMessageTime) < 5*time.Second {
		abuf.append([]byte(e.statusMessage[:messageLen]))
	}
}

func (r *ScreenRenderer) RefreshScreen(e *Editor) {
	e.Scroll()

	var abuf appendBuffer
	abuf.append([]byte(CURSOR_HIDE))
	abuf.append([]byte(CURSOR_HOME))

	r.DrawRows(e, &abuf)
	r.DrawStatusBar(e, &abuf)
	r.DrawMessageBar(e, &abuf)

	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, e.cy-e.rowOffset+1, e.rx-e.colOffset+1))
	abuf.append([]byte(CURSOR_SHOW))

	os.Stdout.Write(abuf.b)
}
