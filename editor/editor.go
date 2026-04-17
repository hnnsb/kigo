package editor

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

/*** helper ***/

// Config Constants
const (
	TAB_STOP               = 4
	CONTROL_SEQUENCE_WIDTH = 2
	QUIT_TIMES             = 2
	MIN_SPLIT_PANE_WIDTH   = 24
	DEFAULT_SHOW_LINE_NUMS = true
)

// getLineEnding returns the appropriate line ending for the current OS
func getLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

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

// Syntax highlighting types
const (
	HL_NORMAL = iota
	HL_COMMENT
	HL_MLCOMMENT
	HL_KEYWORD1
	HL_KEYWORD2
	HL_STRING
	HL_NUMBER
	HL_MATCH
	HL_CONTROL
	HL_UNDERLINE
	HL_BOLD
)

// Syntax highlighting flags
const (
	HL_HIGHLIGHT_NUMBERS = 1 << 0
	HL_HIGHLIGHT_STRINGS = 1 << 1
)

// Config flags
const (
	CFG_ADD_NEWLINE_ON_PARENTHESES = 1 << 0
)

// Editor modes
const (
	EDIT_MODE = iota
	EXPLORER_MODE
	SEARCH_MODE
	SAVE_MODE
	HELP_MODE
)

// Check if the rune is a control character
func isControl(r rune) bool {
	return r < 32 || r == 127
}

// Check if the rune is a digit character
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// Convert a character to its control key equivalent
func withControlKey(c rune) rune {
	return rune(int(c) & 0x1f) // 0x1f is 31 in decimal, which is the control character range
}

// runeIndexOf finds the index of the first occurrence of needle in haystack
func runeIndexOf(haystack, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	if len(needle) > len(haystack) {
		return -1
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		found := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

/*** data ***/

type editorSyntax struct {
	filetype               string
	filematch              []string
	keywords               [][]string
	singlelineCommentStart string
	multilineCommentStart  string
	multilineCommentEnd    string
	indentationChar        rune
	indentationSize        int
	hlFlags                int
	cfgFlags               int
}

type DisplayLine struct {
	idx           int
	chars         []rune
	render        []rune
	hl            []int
	hlOpenComment bool
}

// Buffer holds file content state and related metadata.
type Buffer struct {
	totalRows int
	row       []DisplayLine
	dirty     int
	filename  string
	syntax    *editorSyntax
}

// Viewport holds cursor and visible window state.
type Viewport struct {
	cx, cy     int
	rx         int
	rowOffset  int
	colOffset  int
	screenRows int
	screenCols int
}

type SearchState struct {
	lastMatch   int
	direction   int
	savedHlLine int
	savedHl     []int
}

// Terminal handles terminal-specific operations
type Terminal struct {
	originalState *term.State
}

// Editor represents the text editor state
type Editor struct {
	logger *log.Logger
	Viewport
	Buffer
	statusMessage     string
	statusMessageTime time.Time
	showLineNumbers   bool
	mode              int
	terminal          *Terminal
	renderer          *ScreenRenderer
	activeModal       ModalScreen // Active modal screen (for split view support)
	searchState       SearchState
}

/*** filetypes ***/

var HLDB_ENTRIES = []editorSyntax{
	{
		filetype:  "c",
		filematch: []string{".c", ".h", ".cpp"},
		keywords: [][]string{
			{"switch", "if", "while", "for", "break", "continue", "return", "else",
				"struct", "union", "typedef", "static", "enum", "class", "case"},
			{"int", "long", "double", "float", "char", "unsigned", "signed", "void"},
		},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		indentationChar:        ' ',
		indentationSize:        4,
		hlFlags:                HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
		cfgFlags:               0,
	},
	{
		filetype:  "go",
		filematch: []string{".go", ".mod", ".sum"},
		keywords: [][]string{
			{"break", "case", "chan", "const", "continue", "default", "defer", "else",
				"fallthrough", "for", "go", "goto", "if", "import", "map", "package",
				"range", "return", "select", "struct", "switch", "type", "var"},
			{"interface", "func"},
		},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		indentationChar:        '\t',
		indentationSize:        1,
		hlFlags:                HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
		cfgFlags:               CFG_ADD_NEWLINE_ON_PARENTHESES,
	},
	{
		filetype:               "markdown",
		filematch:              []string{".md", ".markdown"},
		keywords:               [][]string{{"#", "[X]", "[x]", "[ ]", ""}, {"-", "*", "+", "**"}},
		singlelineCommentStart: "#",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		indentationChar:        ' ',
		indentationSize:        2,
		hlFlags:                HL_HIGHLIGHT_NUMBERS,
		cfgFlags:               0,
	},
}

/*** terminal ***/

// Die restores terminal, prints an error message and exits the program
func (e *Editor) Die(format string, args ...any) {
	e.RestoreTerminal()
	os.Stdout.Write([]byte(CLEAR_SCREEN))
	os.Stdout.Write([]byte(CURSOR_HOME))
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// ShowError displays an error message in the status bar instead of terminating
func (e *Editor) ShowError(format string, args ...any) {
	e.SetStatusMessage("Warn: "+format, args...)
}

// Enable raw mode for terminal input.
// This allows us to read every input key and positions the cursor freely
func (e *Editor) EnableRawMode() error {
	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("not running in a terminal")
	}

	fmt.Print(ENTER_ALT_SCREEN) // enter alternate screen

	var err error
	e.terminal.originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return errors.New("enabling terminal raw mode: " + err.Error())
	}
	return nil
}

// Restore the original terminal state, disabling raw mode.
func (e *Editor) RestoreTerminal() {
	fmt.Print(EXIT_ALT_SCREEN) // leave alternate screen

	if e.terminal != nil && e.terminal.originalState != nil {
		term.Restore(int(os.Stdin.Fd()), e.terminal.originalState)
		e.terminal.originalState = nil // Prevent multiple restoration attempts
	}
}

func (e *Editor) readKey() (rune, error) {
	buf := make([]byte, 1)
	n, err := os.Stdin.Read(buf)

	if n != 1 || err != nil {
		return 0, errors.New("reading keyboard input")
	}

	c := buf[0]

	// Handle escape sequences (special keys)
	if c == '\x1b' {
		seq := make([]byte, 5)
		if n, err := os.Stdin.Read(seq[0:2]); n != 2 || err != nil {
			return '\x1b', nil // Return escape if we can't read sequence
		}

		switch seq[0] {
		case '[':
			if seq[1] >= '0' && seq[1] <= '9' {
				if n, err := os.Stdin.Read(seq[2:3]); n != 1 || err != nil {
					return '\x1b', nil
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
						return '\x1b', nil
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
						return '\x1b', nil // For now, we will not handle alt+arrows differently
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
						return '\x1b', nil
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
		return '\x1b', nil // Unknown escape sequence, return escape
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

func getWindowsSize() (int, int, error) {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	return rows, cols, err
}

func (e *Editor) Redraw() {
	var err error
	e.screenRows, e.screenCols, err = getWindowsSize()
	if err != nil {
		e.ShowError("%v", err)
	}
	e.screenRows -= 2 // Adjust for status bar and message bar
	e.RefreshScreen()
}

/*** syntax highlighting ***/

// Check if the character is a separator (whitespace, null, or punctuation)
func isSeparator(c int) bool {
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' || c == 0 {
		return true
	}
	// Check for common programming punctuation separators
	separators := ",.()+-/*=~%<>[];"
	for _, sep := range separators {
		if c == int(sep) {
			return true
		}
	}
	return false
}

func (row *DisplayLine) UpdateSyntax(e *Editor) {
	row.hl = make([]int, len(row.render))

	if e.syntax == nil {
		return
	}

	keywords := e.syntax.keywords

	scs := e.syntax.singlelineCommentStart
	mcs := e.syntax.multilineCommentStart
	mce := e.syntax.multilineCommentEnd
	scsLen := len(scs)
	mcsLen := len(mcs)
	mceLen := len(mce)

	prevSep := true
	var inString rune = 0
	var inComment bool = row.idx > 0 && row.idx-1 < len(e.row) && e.row[row.idx-1].hlOpenComment

	for i := 0; i < len(row.render); {
		c := row.render[i]
		prevHl := HL_NORMAL
		if i > 0 {
			prevHl = row.hl[i-1]
		}

		// Highlight control sequences like ^[ ^A ^B etc.
		if inString == 0 && !inComment && c == '^' && i+1 < len(row.render) {
			row.hl[i] = HL_CONTROL
			row.hl[i+1] = HL_CONTROL

			// TODO: Handle control sequences longer than 2 characters. Do I even print any?
			i += 2
			prevSep = true
			continue
		}

		if scsLen > 0 && inString == 0 && !inComment {
			if strings.HasPrefix(string(row.render[i:]), scs) {
				for j := i; j < len(row.render); j++ {
					row.hl[j] = HL_COMMENT
				}
				break
			}
		}

		if mcsLen > 0 && mceLen > 0 && inString == 0 {
			if inComment {
				// Inside multiline comment color end marker
				row.hl[i] = HL_MLCOMMENT
				if strings.HasPrefix(string(row.render[i:]), mce) {
					for j := range mceLen {
						if i+j < len(row.render) {
							row.hl[i+j] = HL_MLCOMMENT
						} else {
							break // Avoid out of bounds
						}
					}
					inComment = false
					i += mceLen
					continue
				}
				i++ // Continue in the multiline comment
				continue
			} else if strings.HasPrefix(string(row.render[i:]), mcs) {
				// Open multiline comment
				inComment = true
				for j := range mcsLen {
					if i+j < len(row.render) {
						row.hl[i+j] = HL_MLCOMMENT
					} else {
						break // Avoid out of bounds
					}
				}
				i += mcsLen
				continue
			}
		}

		if e.syntax.hlFlags&HL_HIGHLIGHT_STRINGS != 0 {
			if inString != 0 {
				row.hl[i] = HL_STRING
				if c == '\\' && i+1 < len(row.render) {
					row.hl[i+1] = HL_STRING
					i += 2
					continue
				}
				if c == inString {
					inString = 0 // End of string
				}
				i++
				prevSep = true
				continue
			} else {
				if c == '"' || c == '\'' {
					inString = c
					row.hl[i] = HL_STRING
					i++
					continue
				}
			}
		}

		if e.syntax.hlFlags&HL_HIGHLIGHT_NUMBERS != 0 {
			if (isDigit(c) && (prevSep || prevHl == HL_NUMBER)) || (c == '.' && prevHl == HL_NUMBER) {
				row.hl[i] = HL_NUMBER
				i++
				prevSep = false
				continue
			}
		}

		if prevSep {
			// we entered a new word
			for j, sublist := range keywords {
				for _, keyword := range sublist {
					klen := len(keyword)
					if strings.HasPrefix(string(row.render[i:]), keyword) {
						for k := range klen {
							row.hl[i+k] = HL_KEYWORD1 + j
						}
					}
				}
			}
			// No keyword found
			prevSep = false
		} else {
			prevSep = isSeparator(int(c))
		}
		i++
	}

	changed := row.hlOpenComment != inComment
	row.hlOpenComment = inComment
	if changed && row.idx+1 < e.totalRows {
		e.row[row.idx+1].UpdateSyntax(e)
	}
}

func syntaxToGraphics(hl int) (int, int) {
	switch hl {
	case HL_COMMENT, HL_MLCOMMENT:
		return ANSI_COLOR_CYAN, ANSI_ITALIC
	case HL_KEYWORD1:
		return ANSI_COLOR_YELLOW, 0
	case HL_KEYWORD2:
		return ANSI_COLOR_GREEN, 0
	case HL_STRING:
		return ANSI_COLOR_MAGENTA, 0
	case HL_NUMBER:
		return ANSI_COLOR_RED_INTENSE, 0
	case HL_MATCH:
		return ANSI_COLOR_BLUE_INTENSE, ANSI_REVERSE
	case HL_CONTROL:
		return ANSI_COLOR_RED, ANSI_REVERSE
	case HL_BOLD:
		return ANSI_COLOR_CYAN, ANSI_BOLD
	default:
		return ANSI_COLOR_DEFAULT, 0
	}
}

// Get the appropriate reset code for a given style
func getStyleResetCode(style int) int {
	if resetCode, exists := styleResetCodes[style]; exists {
		return resetCode
	}
	return 0
}

func (b *Buffer) SelectSyntaxHighlight(e *Editor) {
	b.syntax = nil
	if b.filename == "" {
		return
	}

	filename := b.filename
	var ext string
	if lastDot := strings.LastIndex(filename, "."); lastDot != -1 {
		ext = filename[lastDot:]
	}

	for j := range HLDB_ENTRIES {
		s := &HLDB_ENTRIES[j]
		for i := range s.filematch {
			pattern := s.filematch[i]

			isExt := pattern[0] == '.'
			if (isExt && ext != "" && ext == pattern) ||
				(!isExt && strings.Contains(filename, pattern)) {
				b.syntax = s

				for filerow := range b.totalRows {
					b.row[filerow].UpdateSyntax(e)
				}
				return
			}
		}
	}
}

func (e *Editor) SelectSyntaxHighlight() {
	e.Buffer.SelectSyntaxHighlight(e)
}

/*** row operations ***/

// Convert cursor X to render X, since rendered characters may differ from original characters (e.g., tabs)
func (row *DisplayLine) cxToRx(cx int) int {
	rx := 0
	for j := range cx {
		if row.chars[j] == '\t' {
			rx += TAB_STOP - (rx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else if isControl(row.chars[j]) {
			rx += CONTROL_SEQUENCE_WIDTH
		} else {
			rx++
		}
	}
	return rx
}

func (row *DisplayLine) rxToCx(rx int) int {
	curRx := 0
	var cx int
	for cx = 0; cx < len(row.chars); cx++ {
		if row.chars[cx] == '\t' {
			curRx += (TAB_STOP - 1) - (curRx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else if isControl(row.chars[cx]) {
			curRx += CONTROL_SEQUENCE_WIDTH
		}
		curRx++

		if curRx > rx {
			return cx
		}
	}
	return cx
}

func (row *DisplayLine) Update(e *Editor) {
	displayWidth := 0

	for _, char := range row.chars {
		if char == '\t' {
			displayWidth += TAB_STOP - (displayWidth % TAB_STOP)
		} else if isControl(char) {
			displayWidth += 2 // ^C representation
		} else {
			displayWidth += 1
		}
	}

	// Allocate render slice with estimated size
	row.render = make([]rune, 0, displayWidth)

	for _, char := range row.chars {
		if char == '\t' {
			// Add spaces until we reach the next TAB_STOP boundary
			row.render = append(row.render, ' ')
			for len(row.render)%TAB_STOP != 0 {
				row.render = append(row.render, ' ')
			}
		} else if isControl(char) {
			row.render = append(row.render, '^')
			switch char {
			case 127: // DEL character
				row.render = append(row.render, '?')
			case '\x1b': // ESC character
				row.render = append(row.render, '[')
			default:
				row.render = append(row.render, char+'@') // Convert control character to printable
			}
		} else {
			row.render = append(row.render, char)
		}
	}

	row.UpdateSyntax(e)
}

func (b *Buffer) InsertRow(e *Editor, at int, s []rune, rowlen int) {
	if at < 0 || at > b.totalRows {
		return
	}

	// Create new row
	newRow := DisplayLine{
		idx:           at,
		chars:         slices.Clone(s[:rowlen]), // Create copy of s with specified length
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Insert row using slice operations
	b.row = append(b.row[:at], append([]DisplayLine{newRow}, b.row[at:]...)...)

	// Update indices for rows that were shifted
	for j := at + 1; j < b.totalRows+1; j++ {
		b.row[j].idx = j
	}

	b.row[at].Update(e)
	b.totalRows++
	b.dirty++
}

func (e *Editor) InsertRow(at int, s []rune, rowlen int) {
	e.Buffer.InsertRow(e, at, s, rowlen)
}

func (b *Buffer) DeleteRow(at int) {
	if at < 0 || at >= b.totalRows {
		return
	}

	// Delete row using slice operations
	b.row = append(b.row[:at], b.row[at+1:]...)

	// Update indices for remaining rows
	for j := at; j < len(b.row); j++ {
		b.row[j].idx = j
	}

	b.totalRows--
	b.dirty++
}

func (e *Editor) DeleteRow(at int) {
	e.Buffer.DeleteRow(at)
}

func (row *DisplayLine) InsertChar(e *Editor, at int, r rune) {
	if at < 0 || at > len(row.chars) {
		at = len(row.chars)
	}

	// Insert rune at position using slices
	row.chars = append(row.chars[:at], append([]rune{r}, row.chars[at:]...)...)

	row.Update(e)
	e.dirty++
}

func (row *DisplayLine) appendRunes(e *Editor, s []rune) {
	row.chars = append(row.chars, s...)

	row.Update(e)
	e.dirty++
}

func (row *DisplayLine) deleteChar(e *Editor, at int) {
	if at < 0 || at >= len(row.chars) {
		return
	}

	// Delete character using slice operations
	row.chars = slices.Delete(row.chars, at, at+1)

	row.Update(e)
	e.dirty++
}

/*** editor operations ***/

func (b *Buffer) InsertRune(e *Editor, v *Viewport, r rune) {
	if v.cy == b.totalRows {
		b.InsertRow(e, b.totalRows, []rune(""), 0)
	}
	b.row[v.cy].InsertChar(e, v.cx, r)
	v.cx++
}

func (e *Editor) InsertRune(r rune) {
	e.Buffer.InsertRune(e, &e.Viewport, r)
}

func (b *Buffer) InsertNewline(e *Editor, v *Viewport) {
	currentRow := &b.row[v.cy]
	currentIndentationRunes := []rune("")
	for _, char := range currentRow.chars {
		if char == ' ' || char == '\t' {
			currentIndentationRunes = append(currentIndentationRunes, char)
		} else {
			break
		}
	}

	if v.cx == 0 {
		b.InsertRow(e, v.cy, currentIndentationRunes, len(currentIndentationRunes))
	} else {
		row := &b.row[v.cy]

		// Insert new row with text from cursor to end of line
		currentIndentationRunes = append(currentIndentationRunes, row.chars[v.cx:]...)
		b.InsertRow(e, v.cy+1, currentIndentationRunes, len(currentIndentationRunes))

		// Truncate current row to text before cursor
		row = &b.row[v.cy]
		row.chars = row.chars[:v.cx]
		row.Update(e)
	}
	v.cy++
	v.cx = 0
}

func (e *Editor) InsertNewline() {
	e.Buffer.InsertNewline(e, &e.Viewport)
}

func (b *Buffer) DeleteChar(e *Editor, v *Viewport) {
	if v.cy == b.totalRows {
		return
	}
	if v.cx == 0 && v.cy == 0 {
		return
	}

	row := &b.row[v.cy]
	if v.cx > 0 {
		row.deleteChar(e, v.cx-1)
		v.cx--
	} else {
		v.cx = len(b.row[v.cy-1].chars)
		b.row[v.cy-1].appendRunes(e, row.chars)
		b.DeleteRow(v.cy) // Delete the current row after appending its content to the previous row
		v.cy--            // Move cursor up to the previous row
	}
}

func (e *Editor) DeleteChar() {
	e.Buffer.DeleteChar(e, &e.Viewport)
}

/*** file i/o ***/

func (b *Buffer) RowsToString() ([]byte, int) {
	var buf strings.Builder
	lineEnding := getLineEnding()

	// Pre-calculate total size for efficiency
	totalSize := 0
	for _, row := range b.row {
		totalSize += len(row.chars) + len(lineEnding) // +len(lineEnding) for line ending
	}
	buf.Grow(totalSize)

	for _, row := range b.row {
		buf.WriteString(string(row.chars))
		buf.WriteString(lineEnding)
	}

	result := buf.String()
	return []byte(result), len(result)
}

func (e *Editor) RowsToString() ([]byte, int) {
	return e.Buffer.RowsToString()
}

func (b *Buffer) Open(e *Editor, filename string) error {
	b.filename = filename
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open file '%s'", filename)
	}
	defer file.Close()

	// Reset buffer state, because we are opening a new file
	b.row = make([]DisplayLine, 0)
	b.totalRows = 0
	b.SelectSyntaxHighlight(e)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing newlines and carriage returns
		for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
			line = line[:len(line)-1]
		}

		runes := []rune(line)
		b.InsertRow(e, b.totalRows, runes, len(runes))
	}

	if err := scanner.Err(); err != nil {
		e.Die("reading file: " + err.Error())
	}
	b.dirty = 0
	return nil
}

func (e *Editor) Open(filename string) error {
	e.cx = 0
	e.cy = 0
	e.rowOffset = 0
	e.colOffset = 0
	e.rx = 0
	e.resetFindState()
	return e.Buffer.Open(e, filename)
}

func (b *Buffer) SaveToFile() (int, error) {
	buf, length := b.RowsToString()

	file, err := os.OpenFile(b.filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	err = file.Truncate(int64(length))
	if err != nil {
		return 0, err
	}

	bytesWritten, err := file.Write(buf)
	if err != nil {
		return 0, err
	}

	if bytesWritten != length {
		return 0, fmt.Errorf("partial write: %d/%d bytes", bytesWritten, length)
	}

	b.dirty = 0
	return length, nil
}

func (e *Editor) Save() {
	if e.filename == "" {
		e.filename = e.Prompt("Save as: %s (ESC to cancel)", nil)
		if e.filename == "" {
			e.SetStatusMessage("Save aborted")
			return
		}
		e.Buffer.SelectSyntaxHighlight(e)
	}

	length, err := e.Buffer.SaveToFile()
	if err != nil {
		e.SetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	e.SetStatusMessage("%d bytes written to disk", length)
}

/*** find ***/
func (e *Editor) FindCallback(query []byte, key int) {
	if e.searchState.savedHl != nil && e.searchState.savedHlLine >= 0 && e.searchState.savedHlLine < len(e.row) {
		copy(e.row[e.searchState.savedHlLine].hl, e.searchState.savedHl)
		e.searchState.savedHl = nil
	}

	switch key {
	case '\r', '\x1b':
		e.searchState.lastMatch = -1
		e.searchState.direction = 1
		return
	case ARROW_RIGHT, ARROW_DOWN:
		e.searchState.direction = 1
	case ARROW_LEFT, ARROW_UP:
		e.searchState.direction = -1
	default:
		e.searchState.lastMatch = -1
		e.searchState.direction = 1
	}

	if e.searchState.lastMatch == -1 {
		e.searchState.direction = 1
	}
	current := e.searchState.lastMatch
	queryRunes := []rune(string(query))

	for range e.totalRows {
		current += e.searchState.direction
		switch current {
		case -1:
			current = e.totalRows - 1
		case e.totalRows:
			current = 0
		}

		row := &e.row[current]
		match := runeIndexOf(row.render, queryRunes)
		if match != -1 {
			e.searchState.lastMatch = current
			e.cy = current
			e.cx = row.rxToCx(match)
			e.rowOffset = e.totalRows

			e.searchState.savedHlLine = current
			e.searchState.savedHl = make([]int, len(row.hl))
			copy(e.searchState.savedHl, row.hl)
			// Highlight the match
			for k := match; k < match+len(queryRunes) && k < len(row.hl); k++ {
				row.hl[k] = HL_MATCH
			}
			break
		}
	}
}

func (e *Editor) Find() {
	savedCx := e.cx
	savedCy := e.cy
	savedColOffset := e.colOffset
	savedRowOffset := e.rowOffset

	query := e.Prompt("Search: %s (Use ESC/Arrows/Enter)", e.FindCallback)

	if query == "" {
		e.cx = savedCx
		e.cy = savedCy
		e.colOffset = savedColOffset
		e.rowOffset = savedRowOffset
	}
}

/*** append buffer ***/

type appendBuffer struct {
	b   []byte
	len int
}

func (ab *appendBuffer) append(s []byte) {
	ab.b = append(ab.b, s...)
	ab.len += len(s)
}

/*** output ***/

func (v *Viewport) Scroll(totalRows int, rows []DisplayLine, contentCols int) {
	v.rx = 0
	if v.cy < totalRows {
		v.rx = rows[v.cy].cxToRx(v.cx)
	}

	if contentCols < 1 {
		contentCols = 1
	}

	if v.cy < v.rowOffset {
		v.rowOffset = v.cy
	}
	if v.cy >= v.rowOffset+v.screenRows {
		v.rowOffset = v.cy - v.screenRows + 1
	}

	if v.rx < v.colOffset {
		v.colOffset = v.rx
	}
	if v.rx >= v.colOffset+contentCols {
		v.colOffset = v.rx - contentCols + 1
	}
}

func (e *Editor) Scroll() {
	e.ensureRenderer()
	contentCols := e.renderer.contentWidthForCurrentView(e)
	if e.mode == EXPLORER_MODE {
		e.scrollExplorerWithPinnedRows(contentCols)
		return
	}
	e.Viewport.Scroll(e.totalRows, e.row, contentCols)
}

func (e *Editor) scrollExplorerWithPinnedRows(contentCols int) {
	e.rx = 0
	if e.cy < e.totalRows {
		e.rx = e.row[e.cy].cxToRx(e.cx)
	}

	if contentCols < 1 {
		contentCols = 1
	}

	if e.rx < e.colOffset {
		e.colOffset = e.rx
	}
	if e.rx >= e.colOffset+contentCols {
		e.colOffset = e.rx - contentCols + 1
	}

	if e.screenRows <= explorerPinnedRows {
		e.rowOffset = 0
		return
	}

	firstScrollableRow := explorerPinnedRows
	visibleScrollableRows := max(e.screenRows-explorerPinnedRows, 1)

	if e.rowOffset < firstScrollableRow {
		e.rowOffset = firstScrollableRow
	}

	if e.cy < firstScrollableRow {
		e.rowOffset = firstScrollableRow
	} else {
		if e.cy < e.rowOffset {
			e.rowOffset = e.cy
		}
		if e.cy >= e.rowOffset+visibleScrollableRows {
			e.rowOffset = e.cy - visibleScrollableRows + 1
		}
	}

	maxRowOffset := max(e.totalRows-visibleScrollableRows, firstScrollableRow)
	if e.rowOffset > maxRowOffset {
		e.rowOffset = maxRowOffset
	}
}

func (e *Editor) DrawRows(abuf *appendBuffer) {
	e.ensureRenderer()
	e.renderer.DrawRows(e, abuf)
}

func (e *Editor) DrawStatusBar(abuf *appendBuffer) {
	e.ensureRenderer()
	e.renderer.DrawStatusBar(e, abuf)
}

func (e *Editor) DrawMessageBar(abuf *appendBuffer) {
	e.ensureRenderer()
	e.renderer.DrawMessageBar(e, abuf)
}

func (e *Editor) RefreshScreen() {
	e.ensureRenderer()
	e.renderer.RefreshScreen(e)
}

func (e *Editor) ensureRenderer() {
	if e.renderer == nil {
		e.renderer = NewScreenRenderer()
	}
}

func (e *Editor) SetStatusMessage(format string, args ...any) {
	e.statusMessage = fmt.Sprintf(format, args...)
	e.statusMessageTime = time.Now()
}

/*** input ***/

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

func (v *Viewport) MoveCursor(key int, totalRows int, rows []DisplayLine) {
	var row *DisplayLine
	if v.cy >= totalRows {
		row = nil
	} else {
		row = &rows[v.cy]
	}

	switch key {
	case ARROW_LEFT:
		if v.cx != 0 {
			v.cx--
		} else if v.cy > 0 {
			v.cy--
			v.cx = len(rows[v.cy].chars)
		}
	case ARROW_RIGHT:
		if row != nil && v.cx < len(row.chars) {
			v.cx++
		} else if row != nil && v.cx == len(row.chars) {
			v.cy++
			v.cx = 0
		}
	case ARROW_UP:
		if v.cy != 0 {
			v.cy--
		}
	case ARROW_DOWN:
		if v.cy < totalRows {
			v.cy++
		}
	case CTRL_ARROW_LEFT:
		if row != nil {
			for v.cx > 0 && isSeparator(int(row.chars[v.cx-1])) {
				v.cx--
			}
			for v.cx > 0 && !isSeparator(int(row.chars[v.cx-1])) {
				v.cx--
			}
		}
	case CTRL_ARROW_RIGHT:
		if row != nil {
			rowlen := len(row.chars)
			for v.cx < rowlen && isSeparator(int(row.chars[v.cx])) {
				v.cx++
			}
			for v.cx < rowlen && !isSeparator(int(row.chars[v.cx])) {
				v.cx++
			}
		}
	}

	if v.cy >= totalRows {
		row = nil
	} else {
		row = &rows[v.cy]
	}
	rowlen := 0
	if row != nil {
		rowlen = len(row.chars)
	}
	if v.cx > rowlen {
		v.cx = rowlen
	}
}

func (e *Editor) MoveCursor(key int) {
	e.Viewport.MoveCursor(key, e.totalRows, e.row)
}

var quitTimes = QUIT_TIMES

func (e *Editor) ProcessKeypress() {
	key, err := e.readKey()
	if err != nil {
		e.ShowError("%v", err)
		return // Skip this keypress and continue
	}

	switch key {
	case HOME_KEY:
		e.cx = 0

	case END_KEY:
		if e.cy < e.totalRows {
			e.cx = len(e.row[e.cy].chars)
		}

	case DELETE_KEY:
		e.MoveCursor(ARROW_RIGHT)
		e.DeleteChar()

	case BACKSPACE: // Handle backspace (127)
		e.DeleteChar()

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

	case CTRL_DELETE: // Forward Delete word
		if e.cy < e.totalRows {
			row := &e.row[e.cy]
			if e.cx < len(row.chars) {
				// Move to the end of the next word
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
			row := &e.row[e.cy]
			if e.cx > 0 {
				// Move to the beginning of the previous word
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

	case SHIFT_ARROW_LEFT, SHIFT_ARROW_RIGHT, SHIFT_ARROW_UP, SHIFT_ARROW_DOWN, SHIFT_HOME, SHIFT_END, SHIFT_PAGE_UP, SHIFT_PAGE_DOWN, CTRL_ARROW_UP, CTRL_ARROW_DOWN, CTRL_HOME, CTRL_END, CTRL_PAGE_UP, CTRL_PAGE_DOWN:
	// Unsupported keys for now - just ignore them

	case CTRL_ALT_ARROW_DOWN: // Move line down
		if e.cy < e.totalRows-1 {
			e.row[e.cy], e.row[e.cy+1] = e.row[e.cy+1], e.row[e.cy]
			e.row[e.cy].idx = e.cy
			e.row[e.cy+1].idx = e.cy + 1
			e.cy++
		}

	case CTRL_ALT_ARROW_UP: // Move line up
		if e.cy > 0 {
			e.row[e.cy], e.row[e.cy-1] = e.row[e.cy-1], e.row[e.cy]
			e.row[e.cy].idx = e.cy
			e.row[e.cy-1].idx = e.cy - 1
			e.cy--
		}

	// Control keys and special characters
	case '\r': // Enter
		row := &e.row[e.cy]

		indentLine := e.cx < len(row.chars) && slices.Contains([]rune{')', '}', ']'}, row.chars[e.cx])
		e.InsertNewline()
		for _, char := range row.chars {
			if char == ' ' || char == '\t' {
				e.cx++
			} else {
				break
			}
		}

		if e.syntax != nil && e.syntax.cfgFlags&CFG_ADD_NEWLINE_ON_PARENTHESES != 0 {
			if indentLine {
				e.InsertNewline()
				e.MoveCursor(ARROW_LEFT)
				for range e.syntax.indentationSize {
					e.InsertRune(e.syntax.indentationChar)
					e.RefreshScreen()

				}
			}
		}

	case '\t':
		// TODO: Better Tab behavior:
		// - indent current line, outdent on shift+tab
		// - dependant on file type (for code indent outdent behavior, for text add character at cursor)
		e.InsertRune('\t')

	case '\x1b': // Escape key
		// Do nothing - just reset quit times

	case withControlKey('q'):
		if e.dirty > 0 && quitTimes > 0 {
			e.SetStatusMessage("WARNING: File has unsaved changes. Press Ctrl-Q %d more times to quit.", quitTimes)
			quitTimes--
			return
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

	case '(':
		e.InsertRune('(')
		e.InsertRune(')')
		e.MoveCursor(ARROW_LEFT)

	case ')':
		if e.cy < e.totalRows {
			row := &e.row[e.cy]
			if e.cx < len(row.chars) && row.chars[e.cx] == ')' {
				e.MoveCursor(ARROW_RIGHT)
			} else {
				e.InsertRune(')')
			}
		}

	case '{':
		e.InsertRune('{')
		e.InsertRune('}')
		e.MoveCursor(ARROW_LEFT)

	case '}':
		if e.cy < e.totalRows {
			row := &e.row[e.cy]
			if e.cx < len(row.chars) && row.chars[e.cx] == '}' {
				e.MoveCursor(ARROW_RIGHT)
			} else {
				e.InsertRune('}')
			}
		}

	case '[':
		e.InsertRune('[')
		e.InsertRune(']')
		e.MoveCursor(ARROW_LEFT)

	case ']':
		if e.cy < e.totalRows {
			row := &e.row[e.cy]
			if e.cx < len(row.chars) && row.chars[e.cx] == ']' {
				e.MoveCursor(ARROW_RIGHT)
			} else {
				e.InsertRune(']')
			}
		}

	case '"':
		e.InsertRune('"')
		e.InsertRune('"')
		e.MoveCursor(ARROW_LEFT)

	case '\'':
		e.InsertRune('\'')
		e.InsertRune('\'')
		e.MoveCursor(ARROW_LEFT)

	default:
		// Insert regular character (including Unicode)
		// Skip control characters except those we explicitly handle
		if !isControl(key) || key >= 128 {
			e.InsertRune(key)
		}
	}

	quitTimes = QUIT_TIMES // Reset quit times after processing a key
}

/*** init ***/

// NewTerminal creates a new Terminal instance
func NewTerminal() *Terminal {
	return &Terminal{}
}

// NewEditor creates a new Editor instance with proper initialization
func NewEditor(logger *log.Logger) Editor {
	return Editor{
		logger:          logger,
		terminal:        NewTerminal(),
		renderer:        NewScreenRenderer(),
		showLineNumbers: DEFAULT_SHOW_LINE_NUMS,
		searchState: SearchState{
			lastMatch:   -1,
			direction:   1,
			savedHlLine: -1,
			savedHl:     nil,
		},
	}
}

func (e *Editor) Init() error {
	e.cx, e.cy = 0, 0
	e.rx = 0
	e.rowOffset = 0
	e.colOffset = 0
	e.totalRows = 0
	e.row = make([]DisplayLine, 0)
	e.dirty = 0
	e.filename = ""
	e.statusMessage = ""
	e.statusMessageTime = time.Time{}
	e.syntax = nil
	e.mode = EDIT_MODE
	e.resetFindState()

	var err error
	e.screenRows, e.screenCols, err = getWindowsSize()
	if err != nil {
		return errors.New("getting window size")
	}
	e.screenRows -= 2
	return nil
}

func (e *Editor) resetFindState() {
	e.searchState.lastMatch = -1
	e.searchState.direction = 1
	e.searchState.savedHlLine = -1
	e.searchState.savedHl = nil
}

func (e *Editor) Debug(format string, args ...any) {
	if e.logger != nil {
		e.logger.Printf("[DEBUG] "+format, args...)
	}
}
