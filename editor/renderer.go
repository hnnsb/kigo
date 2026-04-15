package editor

import (
	"fmt"
	"os"
	"strconv"
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

func lineNumberDigits(totalRows int) int {
	rows := max(totalRows, 1)
	return len(strconv.Itoa(rows))
}

func lineNumberLayout(availableCols int, totalRows int, enabled bool) (int, int) {
	if !enabled || availableCols < 4 {
		return 0, 0
	}

	digits := lineNumberDigits(totalRows)
	prefixWidth := digits + 1 // "<num> "
	if prefixWidth >= availableCols {
		return 0, 0
	}

	return digits, prefixWidth
}

func lineNumbersEnabled(e *Editor) bool {
	return e.showLineNumbers && e.mode != EXPLORER_MODE
}

func appendEmptyLineNumberPrefix(abuf *appendBuffer, digits int) {
	abuf.append(fmt.Appendf(nil, "%*s ", digits, ""))
}

func appendDisplayLine(abuf *appendBuffer, line DisplayLine, startOffset int, maxWidth int, pad bool, lineNum int, lineNumDigits int) {
	if lineNumDigits > 0 {
		abuf.append(fmt.Appendf(nil, "\x1b[%dm%*d\x1b[%dm ", ANSI_COLOR_BLACK_INTENSE, lineNumDigits, lineNum, ANSI_COLOR_DEFAULT))
	}

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

	// Remember: Why pad here?
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

func (r *ScreenRenderer) contentWidthForCurrentView(e *Editor) int {
	availableCols := e.screenCols
	if e.mode == EXPLORER_MODE && e.activeModal != nil {
		if splitViewModal, ok := e.activeModal.(SplitViewModal); ok && splitViewModal.ShouldShowSplitView(e.screenCols) {
			availableCols = e.screenCols / 2
		}
	}

	_, prefixWidth := lineNumberLayout(availableCols, e.totalRows, lineNumbersEnabled(e))
	contentWidth := availableCols - prefixWidth
	if contentWidth < 1 {
		return 1
	}

	return contentWidth
}

func (r *ScreenRenderer) cursorXOffset(e *Editor) int {
	availableCols := e.screenCols
	if e.mode == EXPLORER_MODE && e.activeModal != nil {
		if splitViewModal, ok := e.activeModal.(SplitViewModal); ok && splitViewModal.ShouldShowSplitView(e.screenCols) {
			availableCols = e.screenCols / 2
		}
	}

	_, prefixWidth := lineNumberLayout(availableCols, e.totalRows, lineNumbersEnabled(e))
	return prefixWidth
}

func (r *ScreenRenderer) drawEditorRows(e *Editor, abuf *appendBuffer) {
	lineNumDigits, lineNumPrefixWidth := lineNumberLayout(e.screenCols, e.totalRows, lineNumbersEnabled(e))
	contentWidth := e.screenCols - lineNumPrefixWidth

	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if e.mode == EXPLORER_MODE {
			filerow = explorerFileRowForScreenRow(y, e.rowOffset)
		}
		if filerow >= e.totalRows {
			if lineNumDigits > 0 {
				appendEmptyLineNumberPrefix(abuf, lineNumDigits)
			}

			if e.totalRows == 0 && y == e.screenRows/3 {
				// Welcome Text
				welcome := "KIGO editor -- version " + KIGO_VERSION
				welcomelen := min(len(welcome), max(contentWidth, 0))
				padding := (max(contentWidth, 0) - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte("~"))
					padding--
				}
				for range padding {
					abuf.append([]byte(" "))
				}
				abuf.append([]byte(welcome[:welcomelen]))
			} else {
				if e.mode != EXPLORER_MODE {
					abuf.append([]byte("~"))
				}
			}
		} else {
			appendDisplayLine(abuf, e.row[filerow], e.colOffset, contentWidth, true, e.row[filerow].idx+1, lineNumDigits)
		}
		abuf.append([]byte(CLEAR_LINE))
		abuf.append([]byte("\r\n")) // TODO: Correct, or os specific line ending?
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
	lineNumDigits, lineNumPrefixWidth := lineNumberLayout(leftWidth, e.totalRows, lineNumbersEnabled(e))
	leftContentWidth := leftWidth - lineNumPrefixWidth

	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if e.mode == EXPLORER_MODE {
			filerow = explorerFileRowForScreenRow(y, e.rowOffset)
		}
		if filerow >= e.totalRows {
			if lineNumDigits > 0 {
				appendEmptyLineNumberPrefix(abuf, lineNumDigits)
			}

			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + KIGO_VERSION
				welcomelen := min(len(welcome), max(leftContentWidth, 0))
				padding := (max(leftContentWidth, 0) - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte(" "))
					padding--
				}
				for range padding {
					abuf.append([]byte(" "))
				}
				abuf.append([]byte(welcome[:welcomelen]))
				for i := welcomelen + padding + 1; i < leftContentWidth; i++ {
					abuf.append([]byte(" "))
				}
			} else {
				abuf.append([]byte(" "))
				for i := 1; i < leftContentWidth; i++ {
					abuf.append([]byte(" "))
				}
			}
		} else {
			appendDisplayLine(abuf, e.row[filerow], e.colOffset, leftContentWidth, true, e.row[filerow].idx+1, lineNumDigits)
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
		abuf.append([]byte("\r\n")) // TODO: Correct, or os specific line ending needed?
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

	switch e.mode {
	case EXPLORER_MODE:
		rstatus = fmt.Sprintf("| %d/%d", e.cy-2, len(e.activeModal.(*ExplorerScreen).files))
	default:
		rstatus = fmt.Sprintf("%s | %d/%d", filetype, e.cy+1, e.totalRows)
	}

	abuf.append([]byte(status[:statusLen]))

	rstatusLen := len(rstatus)
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

	cursorCol := e.rx - e.colOffset + r.cursorXOffset(e) + 1
	cursorRow := cursorScreenRow(e)
	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, cursorRow, cursorCol))
	abuf.append([]byte(CURSOR_SHOW))

	os.Stdout.Write(abuf.b)
}

func explorerFileRowForScreenRow(screenRow int, rowOffset int) int {
	if screenRow < explorerPinnedRows {
		return screenRow
	}
	return rowOffset + (screenRow - explorerPinnedRows)
}

func cursorScreenRow(e *Editor) int {
	if e.mode == EXPLORER_MODE {
		if e.cy < explorerPinnedRows {
			return e.cy + 1
		}
		if e.screenRows <= explorerPinnedRows {
			return 1
		}
		if e.rowOffset < explorerPinnedRows {
			return e.cy + 1
		}
		return explorerPinnedRows + (e.cy - e.rowOffset) + 1
	}
	return e.cy - e.rowOffset + 1
}
