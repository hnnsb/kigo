package editor

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/hnnsb/kigo/internal/ansi"
	"golang.org/x/term"
)

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

/*** helper ***/

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
	rows      []DisplayLine
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
	os.Stdout.Write([]byte(ansi.CLEAR_SCREEN))
	os.Stdout.Write([]byte(ansi.CURSOR_HOME))
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

	fmt.Print(ansi.ENTER_ALT_SCREEN) // enter alternate screen

	var err error
	e.terminal.originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return errors.New("enabling terminal raw mode: " + err.Error())
	}
	return nil
}

// Restore the original terminal state, disabling raw mode.
func (e *Editor) RestoreTerminal() {
	fmt.Print(ansi.EXIT_ALT_SCREEN) // leave alternate screen

	if e.terminal != nil && e.terminal.originalState != nil {
		term.Restore(int(os.Stdin.Fd()), e.terminal.originalState)
		e.terminal.originalState = nil // Prevent multiple restoration attempts
	}
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

func (row *DisplayLine) UpdateSyntax(b *Buffer) {
	row.hl = make([]int, len(row.render))

	if b.syntax == nil {
		return
	}

	keywords := b.syntax.keywords

	scs := b.syntax.singlelineCommentStart
	mcs := b.syntax.multilineCommentStart
	mce := b.syntax.multilineCommentEnd
	scsLen := len(scs)
	mcsLen := len(mcs)
	mceLen := len(mce)

	prevSep := true
	var inString rune = 0
	var inComment bool = row.idx > 0 && row.idx-1 < len(b.rows) && b.rows[row.idx-1].hlOpenComment

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

		if b.syntax.hlFlags&HL_HIGHLIGHT_STRINGS != 0 {
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

		if b.syntax.hlFlags&HL_HIGHLIGHT_NUMBERS != 0 {
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
	if changed && row.idx+1 < b.totalRows {
		b.rows[row.idx+1].UpdateSyntax(b)
	}
}

func syntaxToGraphics(hl int) (int, int) {
	switch hl {
	case HL_COMMENT, HL_MLCOMMENT:
		return ansi.COLOR_CYAN, ansi.ITALIC
	case HL_KEYWORD1:
		return ansi.COLOR_YELLOW, 0
	case HL_KEYWORD2:
		return ansi.COLOR_GREEN, 0
	case HL_STRING:
		return ansi.COLOR_MAGENTA, 0
	case HL_NUMBER:
		return ansi.COLOR_RED_INTENSE, 0
	case HL_MATCH:
		return ansi.COLOR_BLUE_INTENSE, ansi.REVERSE
	case HL_CONTROL:
		return ansi.COLOR_RED, ansi.REVERSE
	case HL_BOLD:
		return ansi.COLOR_CYAN, ansi.BOLD
	default:
		return ansi.COLOR_DEFAULT, 0
	}
}

// Get the appropriate reset code for a given style
func getStyleResetCode(style int) int {
	if resetCode, exists := ansi.StyleResetCodes[style]; exists {
		return resetCode
	}
	return 0
}

func (b *Buffer) SelectSyntaxHighlight() {
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
					b.rows[filerow].UpdateSyntax(b)
				}
				return
			}
		}
	}
}

func (e *Editor) SelectSyntaxHighlight() {
	e.Buffer.SelectSyntaxHighlight()
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

func (row *DisplayLine) Update(b *Buffer) {
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
			case ansi.ESC: // ESC character
				row.render = append(row.render, '[')
			default:
				row.render = append(row.render, char+'@') // Convert control character to printable
			}
		} else {
			row.render = append(row.render, char)
		}
	}

	row.UpdateSyntax(b)
}

func (b *Buffer) InsertRow(at int, s []rune, rowlen int) {
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
	b.rows = append(b.rows[:at], append([]DisplayLine{newRow}, b.rows[at:]...)...)

	// Update indices for rows that were shifted
	for j := at + 1; j < b.totalRows+1; j++ {
		b.rows[j].idx = j
	}

	b.rows[at].Update(b)
	b.totalRows++
	b.dirty++
}

func (e *Editor) InsertRow(at int, s []rune, rowlen int) {
	e.Buffer.InsertRow(at, s, rowlen)
}

func (b *Buffer) DeleteRow(at int) {
	if at < 0 || at >= b.totalRows {
		return
	}

	// Delete row using slice operations
	b.rows = append(b.rows[:at], b.rows[at+1:]...)

	// Update indices for remaining rows
	for j := at; j < len(b.rows); j++ {
		b.rows[j].idx = j
	}

	b.totalRows--
	b.dirty++
}

func (e *Editor) DeleteRow(at int) {
	e.Buffer.DeleteRow(at)
}

func (row *DisplayLine) InsertChar(at int, r rune, b *Buffer) {
	if at < 0 || at > len(row.chars) {
		at = len(row.chars)
	}

	// Insert rune at position using slices
	row.chars = append(row.chars[:at], append([]rune{r}, row.chars[at:]...)...)

	row.Update(b)
	b.dirty++
}

func (row *DisplayLine) appendRunes(s []rune, b *Buffer) {
	row.chars = append(row.chars, s...)

	row.Update(b)
	b.dirty++
}

func (row *DisplayLine) deleteChar(at int, b *Buffer) {
	if at < 0 || at >= len(row.chars) {
		return
	}

	// Delete character using slice operations
	row.chars = slices.Delete(row.chars, at, at+1)

	row.Update(b)
	b.dirty++
}

/*** editor operations ***/

func (e *Editor) InsertRune(r rune) {
	if e.cy == e.totalRows {
		e.InsertRow(e.totalRows, []rune(""), 0)
	}
	e.rows[e.cy].InsertChar(e.cx, r, &e.Buffer)
	e.cx++
}

func (e *Editor) InsertNewline() {
	currentRow := &e.rows[e.cy]
	currentIndentationRunes := []rune("")
	for _, char := range currentRow.chars {
		if char == ' ' || char == '\t' {
			currentIndentationRunes = append(currentIndentationRunes, char)
		} else {
			break
		}
	}

	if e.cx == 0 {
		e.InsertRow(e.cy, currentIndentationRunes, len(currentIndentationRunes))
	} else {
		row := &e.rows[e.cy]

		// Insert new row with text from cursor to end of line
		currentIndentationRunes = append(currentIndentationRunes, row.chars[e.cx:]...)
		e.InsertRow(e.cy+1, currentIndentationRunes, len(currentIndentationRunes))

		// Truncate current row to text before cursor
		row = &e.rows[e.cy]
		row.chars = row.chars[:e.cx]
		row.Update(&e.Buffer)
	}
	e.cy++
	e.cx = 0
}

func (e *Editor) DeleteChar() {
	if e.cy == e.totalRows {
		return
	}
	if e.cx == 0 && e.cy == 0 {
		return
	}

	row := &e.rows[e.cy]
	if e.cx > 0 {
		row.deleteChar(e.cx-1, &e.Buffer)
		e.cx--
	} else {
		e.cx = len(e.rows[e.cy-1].chars)
		e.rows[e.cy-1].appendRunes(row.chars, &e.Buffer)
		e.DeleteRow(e.cy) // Delete the current row after appending its content to the previous row
		e.cy--            // Move cursor up to the previous row
	}
}

func (e *Editor) RefreshScreen() {
	e.Scroll()
	e.renderer.RefreshScreen(e)
}

func (e *Editor) SetStatusMessage(format string, args ...any) {
	e.statusMessage = fmt.Sprintf(format, args...)
	e.statusMessageTime = time.Now()
}

func (e *Editor) GetScreenRows() int {
	return e.screenRows
}

func (e *Editor) GetScreenCols() int {
	return e.screenCols
}

func (e *Editor) GetCy() int {
	return e.cy
}

func (e *Editor) SetCy(cy int) {
	e.cy = cy
}

func (e *Editor) GetCx() int {
	return e.cx
}

func (e *Editor) SetCx(cx int) {
	e.cx = cx
}

func (e *Editor) GetRowOffset() int {
	return e.rowOffset
}

func (e *Editor) SetRowOffset(rowOffset int) {
	e.rowOffset = rowOffset
}

func (e *Editor) GetColOffset() int {
	return e.colOffset
}

func (e *Editor) SetColOffset(colOffset int) {
	e.colOffset = colOffset
}

func (e *Editor) GetTotalRows() int {
	return e.totalRows
}

func (e *Editor) GetMode() int {
	return e.mode
}

func (e *Editor) SetMode(mode int) {
	e.mode = mode
}

func (e *Editor) GetDirty() int {
	return e.dirty
}

func (e *Editor) SetRows(rows []DisplayLine) {
	e.rows = rows
}

func (e *Editor) SetTotalRows(totalRows int) {
	e.totalRows = totalRows
}

func (e *Editor) GetRows() []DisplayLine {
	return e.rows
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
	e.rows = make([]DisplayLine, 0)
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
