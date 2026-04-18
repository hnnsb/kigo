package editor

import (
	"fmt"
	"os"
	"slices"
)

var quitTimes = QUIT_TIMES

func (e *Editor) Prompt(prompt string, callback func([]byte, int)) string {
	bufSize := 128
	buf := make([]byte, 0, bufSize)

	for {
		e.SetStatusMessage(prompt, string(buf))
		e.RefreshScreen()

		key, err := e.readKey()
		if err != nil {
			e.ShowError("%v", err)
			continue // Try again instead of terminating
		}

		// Handle special keys and control characters
		switch key {
		case DELETE_KEY, BACKSPACE:
			if len(buf) != 0 {
				buf = buf[:len(buf)-1]
			}
			if callback != nil {
				callback(buf, int(key))
			}

		case '\x1b': // Escape
			e.SetStatusMessage("")
			if callback != nil {
				callback(buf, int(key))
			}
			return ""

		case '\r': // Enter
			if len(buf) != 0 {
				e.SetStatusMessage("")
				if callback != nil {
					callback(buf, int(key))
				}
				return string(buf)
			}

		default:
			// Handle arrow keys for search navigation
			if key == ARROW_LEFT || key == ARROW_RIGHT || key == ARROW_UP || key == ARROW_DOWN {
				if callback != nil {
					callback(buf, int(key))
				}
			} else if !isControl(key) {
				// Regular character input
				runeBytes := []byte(string(key))
				if len(buf)+len(runeBytes) >= bufSize-1 {
					bufSize *= 2
					newBuf := make([]byte, len(buf), bufSize)
					copy(newBuf, buf)
					buf = newBuf
				}
				buf = append(buf, runeBytes...)
				if callback != nil {
					callback(buf, int(key))
				}
			}
		}
	}
}

func (e *Editor) ProcessKeypress() {
	key, err := e.readKey()
	if err != nil {
		e.ShowError("%v", err)
		return // Skip this keypress and continue
	}

	if e.handleInputKey(key) {
		quitTimes = QUIT_TIMES // Reset quit times after processing a key
	}
}

func (e *Editor) handleInputKey(key rune) bool {
	if e.handleNavigationKey(key) {
		return true
	}
	if e.handleDeletionAndLineOps(key) {
		return true
	}
	if handled, resetQuitCounter := e.handleControlCommandKey(key); handled {
		return resetQuitCounter
	}
	if e.handleStructuralEditingKey(key) {
		return true
	}
	if e.handleAutoPairKey(key) {
		return true
	}
	return e.insertRegularKey(key)
}

func (e *Editor) handleNavigationKey(key rune) bool {
	switch key {
	case HOME_KEY:
		e.cx = 0
	case END_KEY:
		if e.cy < e.totalRows {
			e.cx = len(e.rows[e.cy].chars)
		}
	case PAGE_UP:
		e.cy = e.rowOffset
		for range e.screenRows {
			e.MoveCursor(ARROW_UP)
		}
	case PAGE_DOWN:
		e.cy = min(e.rowOffset+e.screenRows-1, e.totalRows)
		for range e.screenRows {
			e.MoveCursor(ARROW_DOWN)
		}
	case ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ARROW_DOWN:
		e.MoveCursor(int(key))
	case CTRL_ARROW_LEFT, CTRL_ARROW_RIGHT:
		e.MoveCursor(int(key))
	default:
		return false
	}

	return true
}

func (e *Editor) handleDeletionAndLineOps(key rune) bool {
	switch key {
	case DELETE_KEY:
		e.MoveCursor(ARROW_RIGHT)
		e.DeleteChar()
	case BACKSPACE: // Handle backspace (127)
		e.DeleteChar()
	case CTRL_DELETE: // Forward Delete word
		if e.cy < e.totalRows {
			row := &e.rows[e.cy]
			if e.cx < len(row.chars) {
				for e.cx < len(row.chars) && !isSeparator(int(row.chars[e.cx])) {
					e.cx++
					e.DeleteChar()
				}
				for e.cx < len(row.chars) && isSeparator(int(row.chars[e.cx])) {
					e.cx++
					e.DeleteChar()
				}
			}
		}
	case withControlKey('h'): // Ctrl+Backspace - Delete previous word
		if e.cy < e.totalRows {
			row := &e.rows[e.cy]
			if e.cx > 0 {
				for e.cx > 0 && isSeparator(int(row.chars[e.cx-1])) {
					e.DeleteChar()
				}
				for e.cx > 0 && !isSeparator(int(row.chars[e.cx-1])) {
					e.DeleteChar()
				}
			}
		}
	case SHIFT_DELETE: // Delete line
		if e.cy < e.totalRows {
			e.cx = 0
			e.DeleteRow(e.cy)
		}
	case CTRL_ALT_ARROW_DOWN: // Move line down
		if e.cy < e.totalRows-1 {
			e.rows[e.cy], e.rows[e.cy+1] = e.rows[e.cy+1], e.rows[e.cy]
			e.rows[e.cy].idx = e.cy
			e.rows[e.cy+1].idx = e.cy + 1
			e.cy++
		}
	case CTRL_ALT_ARROW_UP: // Move line up
		if e.cy > 0 {
			e.rows[e.cy], e.rows[e.cy-1] = e.rows[e.cy-1], e.rows[e.cy]
			e.rows[e.cy].idx = e.cy
			e.rows[e.cy-1].idx = e.cy - 1
			e.cy--
		}
	case SHIFT_ARROW_LEFT, SHIFT_ARROW_RIGHT, SHIFT_ARROW_UP, SHIFT_ARROW_DOWN, SHIFT_HOME, SHIFT_END, SHIFT_PAGE_UP, SHIFT_PAGE_DOWN, CTRL_ARROW_UP, CTRL_ARROW_DOWN, CTRL_HOME, CTRL_END, CTRL_PAGE_UP, CTRL_PAGE_DOWN:
		// Unsupported keys for now - just ignore them
	default:
		return false
	}

	return true
}

func (e *Editor) handleControlCommandKey(key rune) (handled bool, resetQuitCounter bool) {
	switch key {
	case withControlKey('q'):
		if e.dirty > 0 && quitTimes > 0 {
			e.SetStatusMessage("WARNING: File has unsaved changes. Press Ctrl-Q %d more times to quit.", quitTimes)
			quitTimes--
			return true, false
		}
		e.RestoreTerminal()
		fmt.Println("Exited KIGO editor")
		os.Exit(0)
	case withControlKey('s'):
		e.Save()
	case withControlKey('e'):
		e.Explorer()
	case withControlKey('f'):
		e.Find()
	case withControlKey('r'):
		e.Redraw()
	case withControlKey('t'):
		e.Help()
	default:
		return false, false
	}

	return true, true
}

func (e *Editor) handleStructuralEditingKey(key rune) bool {
	switch key {
	case '\r': // Enter
		e.handleEnterKey()
	case '\t':
		// TODO: Better Tab behavior:
		// - indent current line, outdent on shift+tab
		// - dependant on file type (for code indent outdent behavior, for text add character at cursor)
		e.InsertRune('\t')
	case '\x1b': // Escape key
		// Do nothing - just reset quit times
	default:
		return false
	}

	return true
}

func (e *Editor) handleAutoPairKey(key rune) bool {
	switch key {
	case '(':
		e.insertPair('(', ')')
	case ')':
		e.skipOrInsertClosing(')')
	case '{':
		e.insertPair('{', '}')
	case '}':
		e.skipOrInsertClosing('}')
	case '[':
		e.insertPair('[', ']')
	case ']':
		e.skipOrInsertClosing(']')
	case '"':
		e.insertPair('"', '"')
	case '\'':
		e.insertPair('\'', '\'')
	default:
		return false
	}

	return true
}

func (e *Editor) insertRegularKey(key rune) bool {
	// Insert regular character (including Unicode)
	// Skip control characters except those we explicitly handle
	if !isControl(key) || key >= 128 {
		e.InsertRune(key)
		return true
	}

	return false
}

func (e *Editor) handleEnterKey() {
	if len(e.rows) < e.cy+1 {
		e.InsertRow(e.cy, []rune{}, 0)
	}
	row := &e.rows[e.cy]

	indentLine := e.cx < len(row.chars) && slices.Contains([]rune{')', '}', ']'}, row.chars[e.cx])
	e.InsertNewline()
	for _, char := range row.chars {
		if char == ' ' || char == '\t' {
			e.cx++
		} else {
			break
		}
	}

	if e.syntax != nil && e.syntax.cfgFlags&CFG_ADD_NEWLINE_ON_PARENTHESES != 0 && indentLine {
		e.InsertNewline()
		e.MoveCursor(ARROW_LEFT)
		for range e.syntax.indentationSize {
			e.InsertRune(e.syntax.indentationChar)
			e.RefreshScreen()
		}
	}
}

func (e *Editor) insertPair(open rune, close rune) {
	e.InsertRune(open)
	e.InsertRune(close)
	e.MoveCursor(ARROW_LEFT)
}

func (e *Editor) skipOrInsertClosing(close rune) {
	if e.cy < e.totalRows {
		row := &e.rows[e.cy]
		if e.cx < len(row.chars) && row.chars[e.cx] == close {
			e.MoveCursor(ARROW_RIGHT)
			return
		}
	}
	e.InsertRune(close)
}
