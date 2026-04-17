package editor

// ModalScreen represents a modal screen interface that can be displayed in the editor
type ModalScreen interface {
	// GetContent returns the content rows to display
	GetContent() []DisplayLine

	// GetTitle returns the title for the modal screen
	GetTitle() string

	// GetStatusMessage returns the status message for the modal screen
	GetStatusMessage() string

	// HandleKey processes a key press and returns true if the modal should close
	// The second return value indicates whether to restore the previous state (true) or keep current state (false)
	HandleKey(key int, host ModalHost) (bool, bool)

	// Initialize sets up the initial cursor position and any other screen-specific setup
	Initialize(host ModalHost)
}

// ModalHost exposes the editor capabilities required by modal screens.
type ModalHost interface {
	GetScreenRows() int
	GetScreenCols() int
	GetCy() int
	SetCy(cy int)
	GetCx() int
	SetCx(cx int)
	GetRowOffset() int
	SetRowOffset(rowOffset int)
	GetColOffset() int
	SetColOffset(colOffset int)
	GetTotalRows() int
	SetTotalRows(totalRows int)
	GetRows() []DisplayLine
	GetMode() int
	SetMode(mode int)
	GetDirty() int
	SetRows(rows []DisplayLine)
	SetStatusMessage(format string, args ...any)
	ShowError(format string, args ...any)
	Open(filename string) error
}

// SplitViewModal represents a modal that can render as a split view (left content + right preview)
type SplitViewModal interface {
	ModalScreen

	// ShouldShowSplitView determines if the split view should be displayed for the given terminal width
	ShouldShowSplitView(screenCols int) bool

	// GetSplitViewContent returns the left pane content and right pane preview lines
	// rightWidth is the available width for the right pane
	// maxPreviewLines is the maximum number of lines to return
	GetSplitViewContent(rightWidth int, maxPreviewLines int, cursorRow int) (leftContent []DisplayLine, rightPreview []string)
}

// handles the common logic for modal screens
type ModalManager struct {
	savedState EditorState
	screen     ModalScreen
	host       ModalHost
}

// creates a new modal manager
func NewModalManager(editor *Editor, screen ModalScreen) *ModalManager {
	return &ModalManager{
		savedState: editor.getEditorState(),
		screen:     screen,
		host:       editor,
	}
}

// displays the modal screen and handles the interaction loop
func (m *ModalManager) Show(mode int) {
	content := m.screen.GetContent()
	m.setupModalDisplay(content, mode)

	// Store the active modal so split-view modals can be rendered properly
	editor := m.host.(*Editor)
	editor.activeModal = m.screen
	defer func() {
		editor.activeModal = nil
	}()

	// Let the screen initialize itself (e.g., set cursor position)
	m.screen.Initialize(m.host)

	// Main interaction loop
	for {
		editor.RefreshScreen()

		input, err := editor.readKey()
		if err != nil {
			editor.ShowError("%v", err)
			continue
		}

		// The key is now directly a rune, convert to int for screen handlers
		// TODO : Can i just convert to int?
		key := int(input)

		shouldClose, shouldRestore := m.screen.HandleKey(key, m.host)
		if shouldClose {
			if shouldRestore {
				m.restoreState()
			}
			break // Screen requested to close
		}
	}
}

// configures the editor for modal display
func (m *ModalManager) setupModalDisplay(content []DisplayLine, mode int) {
	m.host.SetMode(mode)
	m.host.SetRows(content)
	m.host.SetTotalRows(len(content))
	m.host.SetCx(0)
	m.host.SetCy(0)
	m.host.SetColOffset(0)
	m.host.SetRowOffset(0)
	m.host.SetStatusMessage("%s", m.screen.GetStatusMessage())
}

// restores the editor to its previous state
func (m *ModalManager) restoreState() {
	editor := m.host.(*Editor)
	editor.setEditorState(m.savedState)
	editor.SetStatusMessage("Returned to editor")
}
