package editor

import (
	"fmt"

	"github.com/hnnsb/kigo/internal/version"
)

// HelpScreen implements the ModalScreen interface for the help display
type HelpScreen struct {
	content []DisplayLine
}

// NewHelpScreen creates a new help screen
func NewHelpScreen(editor *Editor) *HelpScreen {
	helpContent := []string{
		"=== KIGO HELP ===",
		"",
		"NAVIGATION:",
		"  Arrow Keys             - Move cursor",
		"  Page Up/Down           - Scroll by page",
		"  Home/End               - Move to line start/end",
		"",
		"EDITING:",
		"  Ctrl+S                 - Save file",
		"  Ctrl+Q                 - Quit (with confirmation if unsaved)",
		"  Delete/Backspace       - Delete characters",
		"  Ctrl+Alt+Arrow Up/Down - Move line up/down",
		"",
		"SEARCH:",
		"  Ctrl+F                 - Find text",
		"  Arrow Up/Down          - Navigate search results",
		"  Escape                 - Cancel search",
		"",
		"FILE OPERATIONS:",
		"  Ctrl+E                 - Open file explorer",
		"",
		"OTHER:",
		"  Ctrl+H                 - Show this help",
		"  Ctrl+R                 - Redraw screen",
		"",
		"About KIGO:",
		fmt.Sprintf("  Version: %s (%s) (%s)", version.Version, version.Commit, version.Date),
		"  A simple terminal-based text editor written in Go",
		"",
		"Press 'q' or Escape to close this help screen.",
	}

	// Convert help content to editor rows
	content := make([]DisplayLine, len(helpContent))
	for i, line := range helpContent {
		content[i] = DisplayLine{
			idx:   i,
			chars: []rune(line),
		}
		content[i].Update(&editor.Buffer)
	}

	return &HelpScreen{
		content: content,
	}
}

// GetContent returns the help content rows
func (h *HelpScreen) GetContent() []DisplayLine {
	return h.content
}

// GetTitle returns the help screen title
func (h *HelpScreen) GetTitle() string {
	return "Help"
}

// GetStatusMessage returns the status message for the help screen
func (h *HelpScreen) GetStatusMessage() string {
	return "Help Screen - Use Arrow Keys to scroll, 'q' or Escape to exit"
}

// Initialize sets up the initial cursor position for the help screen
func (h *HelpScreen) Initialize(host ModalHost) {
	// Help screen starts at the top
	host.SetCy(0)
	host.SetRowOffset(0)
}

// HandleKey processes key presses for the help screen
func (h *HelpScreen) HandleKey(key int, host ModalHost) (bool, bool) {
	switch key {
	case 'q', 'Q', '\x1b': // ESC or 'q' to quit
		return true, true // Close modal and restore previous state

	case ARROW_UP:
		if host.GetCy() > 0 {
			host.SetCy(host.GetCy() - 1)
		} else if host.GetRowOffset() > 0 {
			host.SetRowOffset(host.GetRowOffset() - 1)
		}

	case ARROW_DOWN:
		maxCy := len(h.content) - 1
		screenRows := host.GetScreenRows()
		if host.GetCy() < screenRows-1 && host.GetCy() < maxCy {
			host.SetCy(host.GetCy() + 1)
		} else if host.GetRowOffset()+screenRows < len(h.content) {
			host.SetRowOffset(host.GetRowOffset() + 1)
		}

	case PAGE_UP:
		for i := 0; i < host.GetScreenRows() && (host.GetCy() > 0 || host.GetRowOffset() > 0); i++ {
			if host.GetCy() > 0 {
				host.SetCy(host.GetCy() - 1)
			} else if host.GetRowOffset() > 0 {
				host.SetRowOffset(host.GetRowOffset() - 1)
			}
		}

	case PAGE_DOWN:
		screenRows := host.GetScreenRows()
		for i := 0; i < screenRows && host.GetRowOffset()+host.GetCy() < len(h.content)-1; i++ {
			maxCy := len(h.content) - 1
			if host.GetCy() < screenRows-1 && host.GetCy() < maxCy {
				host.SetCy(host.GetCy() + 1)
			} else if host.GetRowOffset()+screenRows < len(h.content) {
				host.SetRowOffset(host.GetRowOffset() + 1)
			}
		}

	case HOME_KEY:
		host.SetCy(0)
		host.SetRowOffset(0)

	case END_KEY:
		maxRows := len(h.content)
		if maxRows <= host.GetScreenRows() {
			host.SetCy(maxRows - 1)
			host.SetRowOffset(0)
		} else {
			host.SetCy(host.GetScreenRows() - 1)
			host.SetRowOffset(maxRows - host.GetScreenRows())
		}
	}

	return false, false // Don't close modal
}

// Help displays the help screen
func (e *Editor) Help() {
	helpScreen := NewHelpScreen(e)
	modalManager := NewModalManager(e, helpScreen)
	modalManager.Show(HELP_MODE)
}
