package editor

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"unicode/utf8"

	"github.com/hnnsb/kigo/internal/ansi"
)

var quitTimes = QUIT_TIMES

// Key aliases
const (
	BACKSPACE  = 127 // ASCII backspace
	ARROW_LEFT = iota + 1000
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DELETE_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
	CTRL_ARROW_LEFT
	CTRL_ARROW_RIGHT
	CTRL_ARROW_UP
	CTRL_ARROW_DOWN
	CTRL_DELETE
	CTRL_HOME
	CTRL_END
	CTRL_PAGE_UP
	CTRL_PAGE_DOWN
	SHIFT_ARROW_LEFT
	SHIFT_ARROW_RIGHT
	SHIFT_ARROW_UP
	SHIFT_ARROW_DOWN
	SHIFT_DELETE
	SHIFT_HOME
	SHIFT_END
	SHIFT_PAGE_UP
	SHIFT_PAGE_DOWN
	ALT_ARROW_LEFT
	ALT_ARROW_RIGHT
	ALT_ARROW_UP
	ALT_ARROW_DOWN
	ALT_DELETE
	ALT_HOME
	ALT_END
	ALT_PAGE_UP
	ALT_PAGE_DOWN
	CTRL_ALT_ARROW_UP
	CTRL_ALT_ARROW_DOWN
)

/*** helper ***/

// Convert a character to its control key equivalent
func withControlKey(c rune) rune {
	return rune(int(c) & 0x1f) // 0x1f is 31 in decimal, which is the control character range
}

// Check if the rune is a control character
func isControl(r rune) bool {
	return r < 32 || r == 127
}

// Check if the rune is a digit character
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

/*** input ***/

// readKey reads the input sequence from Stdin and returns the corresponding
// rune or constant alias for control sequences
func (e *Editor) readKey() (rune, error) {
	buf := make([]byte, 1)
	n, err := os.Stdin.Read(buf)

	if n != 1 || err != nil {
		return 0, errors.New("reading keyboard input")
	}

	c := buf[0]

	// Handle escape sequences (special keys)
	if c == ansi.ESC {
		seq := make([]byte, 5)
		if n, err := os.Stdin.Read(seq[0:2]); n != 2 || err != nil {
			return ansi.ESC, nil // Return escape if we can't read sequence
		}

		switch seq[0] {
		case '[':
			if seq[1] >= '0' && seq[1] <= '9' {
				if n, err := os.Stdin.Read(seq[2:3]); n != 1 || err != nil {
					return ansi.ESC, nil
				}
				if seq[2] == '~' {
					switch seq[1] {
					case '1', '7':
						return HOME_KEY, nil
					case '3':
						return DELETE_KEY, nil
					case '4', '8':
						return END_KEY, nil
					case '5':
						return PAGE_UP, nil
					case '6':
						return PAGE_DOWN, nil
					}
				}
				if seq[2] == ';' { // Handle modifier keys (e.g., Shift, Ctrl)
					if n, err := os.Stdin.Read(seq[3:5]); n != 2 || err != nil {
						return ansi.ESC, nil
					}
					e.Debug("Modifier key sequence detected: %v - %s", seq, seq)
					switch seq[3] {
					case '2': // Shift
						switch seq[4] {
						case 'A':
							return SHIFT_ARROW_UP, nil
						case 'B':
							return SHIFT_ARROW_DOWN, nil
						case 'C':
							return SHIFT_ARROW_RIGHT, nil
						case 'D':
							return SHIFT_ARROW_LEFT, nil
						case 'H':
							return SHIFT_HOME, nil
						case 'F':
							return SHIFT_END, nil
						case '~':
							switch seq[1] {
							case '3':
								return SHIFT_DELETE, nil
							case '5':
								return SHIFT_PAGE_UP, nil
							case '6':
								return SHIFT_PAGE_DOWN, nil
							}
						}
					case '3': // Alt
						// Alt+arrows are used in Windows Terminal to switch between split windows, so sequences are not sent to kigo.
						return ansi.ESC, nil // For now, we will not handle alt+arrows differently
					case '5': // Ctrl
						switch seq[4] {
						case 'A':
							return CTRL_ARROW_UP, nil
						case 'B':
							return CTRL_ARROW_DOWN, nil
						case 'C':
							return CTRL_ARROW_RIGHT, nil
						case 'D':
							return CTRL_ARROW_LEFT, nil
						case 'H':
							return CTRL_HOME, nil
						case 'F':
							return CTRL_END, nil
						case '~':
							switch seq[1] {
							case '3':
								return CTRL_DELETE, nil
							case '5':
								return CTRL_PAGE_UP, nil
							case '6':
								return CTRL_PAGE_DOWN, nil
							}
						}
					case '6': // Ctrl+Shift
						return ansi.ESC, nil
					case '7': // Ctrl+Alt
						switch seq[4] {
						case 'A':
							return CTRL_ALT_ARROW_UP, nil
						case 'B':
							return CTRL_ALT_ARROW_DOWN, nil
						}
					}
				}
			} else {
				switch seq[1] {
				case 'A':
					return ARROW_UP, nil
				case 'B':
					return ARROW_DOWN, nil
				case 'C':
					return ARROW_RIGHT, nil
				case 'D':
					return ARROW_LEFT, nil
				case 'H':
					return HOME_KEY, nil
				case 'F':
					return END_KEY, nil
				}
			}
		case 'O':
			switch seq[1] {
			case 'H':
				return HOME_KEY, nil
			case 'F':
				return END_KEY, nil
			}
		}
		return ansi.ESC, nil // Unknown escape sequence, return escape
	}

	// For ASCII characters, return directly
	if c < 128 {
		e.Debug("Read character %v - %c", c, c)
		return rune(c), nil
	}

	// Handle multi-byte UTF-8 characters
	// Put the first byte back and read the full UTF-8 sequence
	var utfBuf [4]byte
	utfBuf[0] = c

	// Determine how many more bytes we need
	var totalBytes int
	if c&0xE0 == 0xC0 {
		totalBytes = 2
	} else if c&0xF0 == 0xE0 {
		totalBytes = 3
	} else if c&0xF8 == 0xF0 {
		totalBytes = 4
	} else {
		return utf8.RuneError, errors.New("invalid UTF-8 sequence")
	}

	// Read remaining bytes
	if totalBytes > 1 {
		n, err := os.Stdin.Read(utfBuf[1:totalBytes])
		if n != totalBytes-1 || err != nil {
			return utf8.RuneError, errors.New("reading UTF-8 sequence")
		}
	}

	// Decode UTF-8
	r, size := utf8.DecodeRune(utfBuf[:totalBytes])
	if r == utf8.RuneError || size != totalBytes {
		return utf8.RuneError, errors.New("invalid UTF-8 character")
	}

	return r, nil
}

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

		case ansi.ESC: // Escape
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

// ProcessKeypress reads an input sequence and initiates the correct editor action
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
		e.skipOrInsert(')')
	case '{':
		e.insertPair('{', '}')
	case '}':
		e.skipOrInsert('}')
	case '[':
		e.insertPair('[', ']')
	case ']':
		e.skipOrInsert(']')
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

	// Add extra newline and indent if inside brackets
	if e.syntax != nil && e.syntax.cfgFlags&CFG_ADD_NEWLINE_ON_PARENTHESES != 0 && indentLine {
		e.InsertNewline()
		e.MoveCursor(ARROW_LEFT)
		for range e.syntax.indentationSize {
			e.InsertRune(e.syntax.indentationChar)
		}
	}
}

// insertPair inserts the rune pair and positions the cursor inside the pair.
func (e *Editor) insertPair(open rune, close rune) {
	e.InsertRune(open)
	e.InsertRune(close)
	e.MoveCursor(ARROW_LEFT)
}

// skipOrInsert inserts the given rune r, only if the next character is not
// the same.
func (e *Editor) skipOrInsert(r rune) {
	if e.cy < e.totalRows {
		row := &e.rows[e.cy]
		if e.cx < len(row.chars) && row.chars[e.cx] == r {
			e.MoveCursor(ARROW_RIGHT)
			return
		}
	}
	e.InsertRune(r)
}
