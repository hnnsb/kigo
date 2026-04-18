package ansi

import "fmt"

const (
	// Escape
	ESC = '\x1b'
	// Control Sequence Introducer
	CSI = string(ESC) + "["
)

// ANSI escape sequences for terminal control
const (
	// Screen control
	CLEAR_SCREEN     = "\x1b[2J"     // Clear entire screen
	CLEAR_LINE       = "\x1b[K"      // Clear line from cursor to end
	CURSOR_HOME      = "\x1b[H"      // Move cursor to top-left (1,1)
	ENTER_ALT_SCREEN = "\x1b[?1049h" // Switch to alternate screen buffer
	EXIT_ALT_SCREEN  = "\x1b[?1049l" // Switch back to normal screen buffer

	// Cursor visibility
	CURSOR_HIDE = "\x1b[?25l" // Hide cursor
	CURSOR_SHOW = "\x1b[?25h" // Show cursor

	// Cursor positioning
	CURSOR_BOTTOM_RIGHT = "\x1b[999;999H" // Move cursor to bottom-right corner
	CURSOR_GET_POSITION = "\x1b[6n"       // Request cursor position

	// Format strings for dynamic positioning
	CURSOR_POSITION_FORMAT = "\x1b[%d;%dH" // Format for moving cursor to specific row;col
	CURSOR_RESPONSE_FORMAT = "\x1b[%d;%dR" // Format for parsing cursor position response

	// Text formatting
	COLORS_RESET  = "\x1b[m"
	COLORS_INVERT = "\x1b[7m"
)

// ANSI Graphics Mode Constants
const (
	RESET_ALL     = 0
	BOLD          = 1
	DIM           = 2
	ITALIC        = 3
	UNDERLINE     = 4
	BLINK         = 5
	REVERSE       = 7
	STRIKETHROUGH = 9

	// Reset codes for specific styles
	RESET_BOLD          = 22
	RESET_DIM           = 22
	RESET_ITALIC        = 23
	RESET_UNDERLINE     = 24
	RESET_BLINK         = 25
	RESET_REVERSE       = 27
	RESET_STRIKETHROUGH = 29

	// Color codes
	COLOR_BLACK   = 30
	COLOR_RED     = 31
	COLOR_GREEN   = 32
	COLOR_YELLOW  = 33
	COLOR_BLUE    = 34
	COLOR_MAGENTA = 35
	COLOR_CYAN    = 36
	COLOR_WHITE   = 37
	COLOR_DEFAULT = 39

	BG_COLOR_BLACK   = 40
	BG_COLOR_RED     = 41
	BG_COLOR_GREEN   = 42
	BG_COLOR_YELLOW  = 43
	BG_COLOR_BLUE    = 44
	BG_COLOR_MAGENTA = 45
	BG_COLOR_CYAN    = 46
	BG_COLOR_WHITE   = 47
	BG_COLOR_DEFAULT = 49

	COLOR_BLACK_INTENSE   = 90
	COLOR_RED_INTENSE     = 91
	COLOR_GREEN_INTENSE   = 92
	COLOR_YELLOW_INTENSE  = 93
	COLOR_BLUE_INTENSE    = 94
	COLOR_MAGENTA_INTENSE = 95
	COLOR_CYAN_INTENSE    = 96
	COLOR_WHITE_INTENSE   = 97
)

// Style reset lookup table
var StyleResetCodes = map[int]int{
	BOLD:          RESET_BOLD,
	DIM:           RESET_DIM,
	ITALIC:        RESET_ITALIC,
	UNDERLINE:     RESET_UNDERLINE,
	BLINK:         RESET_BLINK,
	REVERSE:       RESET_REVERSE,
	STRIKETHROUGH: RESET_STRIKETHROUGH,
	0:             0, // Normal style has no reset needed
}

// escGraphi creates the control sequence, consisting of control sequence
// introducer (CSI) and style, to apply the given style modifier:
//	 "\x1b[<style>m"
func EscGraphic(style int) []byte {
	return fmt.Appendf(nil, "%s%dm", CSI, style)
}
