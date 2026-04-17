package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hnnsb/kigo/internal/version"
	"github.com/mattn/go-runewidth"
)

// ScreenRenderer is responsible for drawing editor state to the terminal.
type ScreenRenderer struct{}

type appendBuffer struct {
	b   []byte
	len int
}

func (ab *appendBuffer) append(s []byte) {
	ab.b = append(ab.b, s...)
	ab.len += len(s)
}

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

func lineNumbersEnabled(showLineNumbers bool, mode int) bool {
	return showLineNumbers && mode != EXPLORER_MODE
}

func appendEmptyLineNumberPrefix(abuf *appendBuffer, digits int) {
	abuf.append(fmt.Appendf(nil, "%*s ", digits, ""))
}

func appendDisplayLine(abuf *appendBuffer, line DisplayLine, startOffset int, maxWidth int, pad bool, lineNum int, lineNumDigits int, cursorRow int) {
	if lineNumDigits > 0 {
		color := ANSI_COLOR_BLACK_INTENSE
		if line.idx == cursorRow {
			color = ANSI_COLOR_DEFAULT
		}
		abuf.append(fmt.Appendf(nil, "\x1b[%dm%*d\x1b[%dm ", color, lineNumDigits, lineNum, ANSI_COLOR_DEFAULT))
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

func (r *ScreenRenderer) cursorXOffset(e *Editor, availableCols int) int {
	_, prefixWidth := lineNumberLayout(availableCols, e.totalRows, lineNumbersEnabled(e.showLineNumbers, e.mode))
	return prefixWidth
}

func (r *ScreenRenderer) drawEditorRows(e *Editor, abuf *appendBuffer, availableCols int) {
	lineNumDigits, lineNumPrefixWidth := lineNumberLayout(availableCols, e.totalRows, lineNumbersEnabled(e.showLineNumbers, e.mode))
	contentWidth := availableCols - lineNumPrefixWidth

	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if e.mode == EXPLORER_MODE {
			filerow = explorerFileRowForScreenRow(y, e.rowOffset, explorerPinnedRows)
		}
		if filerow >= e.totalRows {
			if lineNumDigits > 0 {
				appendEmptyLineNumberPrefix(abuf, lineNumDigits)
			}

			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + version.Version
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
			appendDisplayLine(abuf, e.rows[filerow], e.colOffset, contentWidth, true, e.rows[filerow].idx+1, lineNumDigits, e.cy)
		}
		abuf.append([]byte(CLEAR_LINE))
		abuf.append([]byte("\r\n")) // TODO: Correct, or os specific line ending?
	}
}

func (r *ScreenRenderer) DrawRows(e *Editor, abuf *appendBuffer, splitViewEnabled bool, leftWidth int, rightWidth int, rightPreview []string) {
	if splitViewEnabled {
		r.drawSplitViewRows(e, rightPreview, leftWidth, rightWidth, abuf)
		return
	}
	r.drawEditorRows(e, abuf, e.screenCols)
}

func (r *ScreenRenderer) drawSplitViewRows(e *Editor, rightPreview []string, leftWidth int, rightWidth int, abuf *appendBuffer) {
	if rightWidth < MIN_SPLIT_PANE_WIDTH {
		r.drawEditorRows(e, abuf, e.screenCols)
		return
	}

	lineNumDigits, lineNumPrefixWidth := lineNumberLayout(leftWidth, e.totalRows, lineNumbersEnabled(e.showLineNumbers, e.mode))
	leftContentWidth := leftWidth - lineNumPrefixWidth

	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if e.mode == EXPLORER_MODE {
			filerow = explorerFileRowForScreenRow(y, e.rowOffset, explorerPinnedRows)
		}
		if filerow >= e.totalRows {
			if lineNumDigits > 0 {
				appendEmptyLineNumberPrefix(abuf, lineNumDigits)
			}

			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + version.Version
				welcomelen := min(len(welcome), max(leftContentWidth, 0))
				padding := (max(leftContentWidth, 0) - welcomelen) / 2
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
				} else {
					for range max(leftContentWidth, 0) {
						abuf.append([]byte(" "))
					}
				}
			}
		} else {
			appendDisplayLine(abuf, e.rows[filerow], e.colOffset, leftContentWidth, true, e.rows[filerow].idx+1, lineNumDigits, e.cy)
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
		filename = filepath.Base(e.filename)
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
	if e.syntax != nil && e.syntax.filetype != "" {
		filetype = e.syntax.filetype
	}

	switch e.mode {
	case EXPLORER_MODE:
		filecount := 0
		if explorer, ok := e.activeModal.(*ExplorerScreen); ok {
			filecount = len(explorer.files)
		}
		rstatus = fmt.Sprintf("| %d/%d", e.cy-2, filecount)
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
	abuf.append([]byte("\r\n")) // TODO: Correct, or os specific line ending?
}

func (r *ScreenRenderer) DrawMessageBar(e *Editor, abuf *appendBuffer) {
	abuf.append([]byte(CLEAR_LINE))
	messageLen := min(len(e.statusMessage), e.screenCols)
	if time.Since(e.statusMessageTime) < 5*time.Second {
		abuf.append([]byte(e.statusMessage[:messageLen]))
	}
}

func (r *ScreenRenderer) RefreshScreen(e *Editor) {
	leftWidth, rightWidth, splitViewEnabled, rightPreview := r.splitViewState(e)

	var abuf appendBuffer
	abuf.append([]byte(CURSOR_HIDE))
	abuf.append([]byte(CURSOR_HOME))

	r.DrawRows(e, &abuf, splitViewEnabled, leftWidth, rightWidth, rightPreview)
	r.DrawStatusBar(e, &abuf)
	r.DrawMessageBar(e, &abuf)

	availableCols := e.screenCols
	if splitViewEnabled {
		availableCols = leftWidth
	}
	cursorCol := e.rx - e.colOffset + r.cursorXOffset(e, availableCols) + 1
	cursorRow := cursorScreenRow(e)
	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, cursorRow, cursorCol))
	abuf.append([]byte(CURSOR_SHOW))

	os.Stdout.Write(abuf.b)
}

func (r *ScreenRenderer) splitViewState(e *Editor) (leftWidth int, rightWidth int, enabled bool, rightPreview []string) {
	leftWidth, rightWidth, enabled = r.splitViewWidths(e)
	if !enabled {
		return leftWidth, rightWidth, false, nil
	}

	splitViewModal := e.activeModal.(SplitViewModal)
	_, rightPreview = splitViewModal.GetSplitViewContent(rightWidth, e.screenRows, e.cy)
	return leftWidth, rightWidth, true, rightPreview
}

func (r *ScreenRenderer) splitViewWidths(e *Editor) (leftWidth int, rightWidth int, showSplit bool) {
	if e.mode != EXPLORER_MODE || e.activeModal == nil {
		return 0, 0, false
	}

	leftWidth = e.screenCols / 2
	rightWidth = e.screenCols - leftWidth - 1
	if rightWidth < MIN_SPLIT_PANE_WIDTH {
		return leftWidth, rightWidth, false
	}

	splitViewModal, ok := e.activeModal.(SplitViewModal)
	if !ok {
		return leftWidth, rightWidth, false
	}

	return leftWidth, rightWidth, splitViewModal.ShouldShowSplitView(e.screenCols)
}

func explorerFileRowForScreenRow(screenRow int, rowOffset int, pinnedRows int) int {
	if screenRow < pinnedRows {
		return screenRow
	}
	return rowOffset + (screenRow - pinnedRows)
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

func (r *ScreenRenderer) contentWidthForCurrentView(e *Editor) int {
	availableCols := e.screenCols
	if e.mode == EXPLORER_MODE && e.activeModal != nil {
		if splitViewModal, ok := e.activeModal.(SplitViewModal); ok && splitViewModal.ShouldShowSplitView(e.screenCols) {
			availableCols = e.screenCols / 2
		}
	}

	_, prefixWidth := lineNumberLayout(availableCols, e.totalRows, lineNumbersEnabled(e.showLineNumbers, e.mode))
	contentWidth := availableCols - prefixWidth
	if contentWidth < 1 {
		return 1
	}

	return contentWidth
}
