package editor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	minExplorerPreviewWidth = 90
	previewPaneMinWidth     = 24
	previewReadByteLimit    = 64 * 1024
	binarySampleSize        = 512
)

// EditorState represents the saved state of the editor
type EditorState struct {
	rows      []DisplayLine
	totalRows int
	cx, cy    int
	colOffset int
	rowOffset int
}

// getEditorState creates a snapshot of the current editor state
func (e *Editor) getEditorState() EditorState {
	return EditorState{
		rows:      e.row,
		totalRows: e.totalRows,
		cx:        e.cx,
		cy:        e.cy,
		colOffset: e.colOffset,
		rowOffset: e.rowOffset,
	}
}

// setEditorState restores the editor to a previously saved state
func (e *Editor) setEditorState(state EditorState) {
	e.row = state.rows
	e.totalRows = state.totalRows
	e.cx = state.cx
	e.cy = state.cy
	e.colOffset = state.colOffset
	e.rowOffset = state.rowOffset
	e.mode = EDIT_MODE
}

func (ex *ExplorerScreen) separatorRowIndex() int {
	return 1
}

func (ex *ExplorerScreen) parentRowIndex() int {
	return 2
}

func (ex *ExplorerScreen) firstFileRowIndex() int {
	if ex.hasParentDir {
		return 3
	}
	return 2
}

func (ex *ExplorerScreen) rowIndexForFile(index int) int {
	return ex.firstFileRowIndex() + index
}

func (ex *ExplorerScreen) setExplorerContent(e *Editor) {
	e.row = ex.content
	e.totalRows = len(ex.content)
	e.rowOffset = 0
	e.SetStatusMessage("%s", ex.GetStatusMessage())
}

func (ex *ExplorerScreen) setCursorToFirstFile(e *Editor) {
	e.cy = ex.firstFileRowIndex()
	ex.setExplorerContent(e)
}

func (ex *ExplorerScreen) selectionAtCursor(e *Editor) (selectedPath string, selectedEntry os.DirEntry, hasSelection bool) {
	if ex.hasParentDir && e.cy == ex.parentRowIndex() {
		return filepath.Dir(ex.currentDir), nil, true
	}

	firstFileRow := ex.firstFileRowIndex()
	if e.cy < firstFileRow {
		return "", nil, false
	}

	fileIndex := e.cy - firstFileRow
	if fileIndex < 0 || fileIndex >= len(ex.files) {
		return "", nil, false
	}

	entry := ex.files[fileIndex]
	return filepath.Join(ex.currentDir, entry.Name()), entry, true
}

// ExplorerScreen implements the ModalScreen interface for file exploration
type ExplorerScreen struct {
	currentDir   string
	files        []os.DirEntry
	hasParentDir bool
	content      []DisplayLine
	editor       *Editor
}

// NewExplorerScreen creates a new explorer screen
func NewExplorerScreen(editor *Editor, startDir string) *ExplorerScreen {
	explorer := &ExplorerScreen{
		currentDir: startDir,
		editor:     editor,
	}
	err := explorer.refreshContent()
	if err != nil {
		editor.ShowError("Failed to read directory: %v", err)
		return nil
	}
	return explorer
}

// refreshContent updates the explorer content for the current directory
func (ex *ExplorerScreen) refreshContent() error {
	// Read current directory contents
	files, err := os.ReadDir(ex.currentDir)
	slices.SortFunc(files, func(a, b os.DirEntry) int {
		// Directories first
		if a.IsDir() && !b.IsDir() {
			return -1
		}
		if !a.IsDir() && b.IsDir() {
			return 1
		}
		return strings.Compare(a.Name(), b.Name())
	})

	if err != nil {
		return err
	}

	ex.files = files
	ex.hasParentDir = ex.currentDir != "." && ex.currentDir != "/"

	// Create content rows
	ex.content = ex.createExplorerRows(files, ex.currentDir)

	return nil
}

// createExplorerRows creates all the display rows for the file explorer
func (ex *ExplorerScreen) createExplorerRows(files []os.DirEntry, currentDir string) []DisplayLine {
	explorerRows := make([]DisplayLine, 0, len(files)+2)

	// Add header
	headerText := fmt.Sprintf("File Explorer: %s ", currentDir)
	headerRow := DisplayLine{
		idx:   0,
		chars: []rune(headerText),
	}
	headerRow.Update(ex.editor)
	for i := range headerRow.hl {
		headerRow.hl[i] = HL_NORMAL
	}
	explorerRows = append(explorerRows, headerRow)
	separatorRow := DisplayLine{idx: ex.separatorRowIndex(), chars: []rune(strings.Repeat("-", ex.editor.screenCols))}
	separatorRow.Update(ex.editor)
	explorerRows = append(explorerRows, separatorRow)

	// Add parent directory option (unless we're at root)
	if ex.hasParentDir {
		parentText := "⏎ .."
		parentRow := DisplayLine{
			idx:   ex.parentRowIndex(),
			chars: []rune(parentText),
		}
		parentRow.Update(ex.editor)
		explorerRows = append(explorerRows, parentRow)
	}

	// Add files
	for i, file := range files {
		fileRow := ex.createFileDisplayRow(i, file)
		fileRow.Update(ex.editor)
		explorerRows = append(explorerRows, fileRow)
	}

	return explorerRows
}

// createFileDisplayRow creates a formatted display row for a file or directory
func (ex *ExplorerScreen) createFileDisplayRow(index int, file os.DirEntry) DisplayLine {
	var fileInfo string
	if file.IsDir() {
		fileInfo = fmt.Sprintf("🗀 %s/", file.Name())
	} else {
		info, _ := file.Info()
		size := ""
		if info != nil {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		fileInfo = fmt.Sprintf("🗋 %s%s", file.Name(), size)
	}

	return DisplayLine{
		idx:   ex.rowIndexForFile(index),
		chars: []rune(fileInfo),
	}
}

// GetContent returns the explorer content rows
func (ex *ExplorerScreen) GetContent() []DisplayLine {
	return ex.content
}

// GetTitle returns the explorer screen title
func (ex *ExplorerScreen) GetTitle() string {
	return "File Explorer"
}

// GetStatusMessage returns the status message for the explorer screen
func (ex *ExplorerScreen) GetStatusMessage() string {
	return fmt.Sprintf("File Explorer: %s - %d items (↑↓/←→=Navigate, ESC/q=quit)", ex.currentDir, len(ex.files))
}

// ShouldShowSplitView determines if the split view should be displayed.
func (ex *ExplorerScreen) ShouldShowSplitView(screenCols int) bool {
	return screenCols >= minExplorerPreviewWidth
}

// GetSplitViewContent returns the explorer file list and file preview for split view rendering.
func (ex *ExplorerScreen) GetSplitViewContent(e *Editor, rightWidth int, maxPreviewLines int) ([]DisplayLine, []string) {
	// Left pane: explorer content (file list)
	leftContent := ex.GetContent()

	// Right pane: file preview
	rightPreview := ex.buildPreviewLines(e, rightWidth, maxPreviewLines)

	return leftContent, rightPreview
}

// isBinaryFile checks if a file is likely binary by looking for null bytes and control characters.
func isBinaryFile(filepath string) bool {
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, binarySampleSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (strong indicator of binary)
	for i := range n {
		if buf[i] == 0 {
			return true
		}
	}

	// Count control characters (excluding common ones like \n, \r, \t)
	controlCount := 0
	for i := range n {
		b := buf[i]
		// Allow: tab (9), newline (10), carriage return (13), and printable ASCII (32-126)
		if (b < 32 && b != 9 && b != 10 && b != 13) || b == 127 {
			controlCount++
		}
	}

	// If more than 30% of sampled bytes are control characters, likely binary
	return controlCount > (n / 3)
}

// sanitizePreviewText removes control sequences and non-printable characters for safe display.
func sanitizePreviewText(text string) string {
	var result strings.Builder
	for _, r := range text {
		// Allow printable characters, common whitespace, and Unicode
		if (r >= 32 && r <= 126) || r == '\t' || r >= 128 {
			result.WriteRune(r)
		} else if r == '\n' || r == '\r' {
			// Skip newlines/returns; they're handled by line breaks
		} else {
			// Replace other control characters with •
			result.WriteRune('•')
		}
	}
	return result.String()
}

// buildPreviewLines returns text lines to render in the preview pane.
func (ex *ExplorerScreen) buildPreviewLines(e *Editor, width int, maxLines int) []string {
	if width < previewPaneMinWidth || maxLines <= 0 {
		return nil
	}
	preview := make([]string, 0, maxLines)
	selectedPath, selectedEntry, hasSelection := ex.selectionAtCursor(e)

	// Build header and separator
	header := ex.buildPreviewHeader(selectedPath, selectedEntry, hasSelection)
	preview = appendPreviewEntry(preview, header, width, maxLines)
	preview = appendPreviewEntry(preview, strings.Repeat("-", width), width, maxLines)

	// Build content based on selection type
	if !hasSelection {
		return trimPreviewLines(preview, maxLines, width)
	}

	if selectedEntry == nil {
		// Parent directory - show nothing below header
		return trimPreviewLines(preview, maxLines, width)
	}

	if selectedEntry.IsDir() {
		preview = ex.previewDirectory(preview, selectedPath, width, maxLines)
	} else {
		preview = ex.previewFile(preview, selectedPath, width, maxLines)
	}
	return trimPreviewLines(preview, maxLines, width)
}

// buildPreviewHeader creates the header line for the preview pane.
func (ex *ExplorerScreen) buildPreviewHeader(selectedPath string, selectedEntry os.DirEntry, hasSelection bool) string {
	if !hasSelection {
		return "No selection"
	}

	if selectedEntry == nil {
		// Parent directory case
		return fmt.Sprintf("Preview: %s/", filepath.Base(selectedPath))
	}

	name := selectedEntry.Name()
	if selectedEntry.IsDir() {
		children, err := os.ReadDir(selectedPath)
		if err != nil {
			return fmt.Sprintf("Preview: %s/", name)
		}
		return fmt.Sprintf("Preview: %s/ (%d items)", name, len(children))
	}

	return fmt.Sprintf("Preview: %s", name)
}

// previewDirectory appends directory content lines to the preview.
func (ex *ExplorerScreen) previewDirectory(preview []string, selectedPath string, width int, maxLines int) []string {
	children, err := os.ReadDir(selectedPath)
	if err != nil {
		return appendPreviewEntry(preview, fmt.Sprintf("Error: %v", err), width, maxLines)
	}

	for _, child := range children {
		if len(preview) >= maxLines {
			break
		}
		prefix := "🗋 "
		if child.IsDir() {
			prefix = "🗀 "
		}
		preview = appendPreviewEntry(preview, prefix+child.Name(), width, maxLines)
	}
	return preview
}

// previewFile appends file content lines to the preview.
func (ex *ExplorerScreen) previewFile(preview []string, selectedPath string, width int, maxLines int) []string {
	if isBinaryFile(selectedPath) {
		return appendPreviewEntry(preview, "(binary file - preview not available)", width, maxLines)
	}

	file, err := os.Open(selectedPath)
	if err != nil {
		return appendPreviewEntry(preview, fmt.Sprintf("Error opening file: %v", err), width, maxLines)
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, previewReadByteLimit)
	scanner := bufio.NewScanner(limitedReader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() && len(preview) < maxLines {
		preview = appendPreviewEntry(preview, sanitizePreviewText(scanner.Text()), width, maxLines)
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		preview = appendPreviewEntry(preview, fmt.Sprintf("Read error: %v", err), width, maxLines)
	}

	if lineCount == 0 && len(preview) < maxLines {
		preview = appendPreviewEntry(preview, "(empty file)", width, maxLines)
	}

	if len(preview) >= maxLines {
		preview[maxLines-1] = fitPreviewLine("...", width)
	}

	return preview
}

func appendPreviewEntry(lines []string, text string, width int, maxLines int) []string {
	if len(lines) >= maxLines {
		return lines
	}
	return append(lines, fitPreviewLine(text, width))
}

func trimPreviewLines(lines []string, maxLines int, width int) []string {
	if len(lines) > maxLines {
		return lines[:maxLines]
	}
	// Pad with empty lines to always have exactly maxLines
	emptyLine := strings.Repeat(" ", width)
	for len(lines) < maxLines {
		lines = append(lines, emptyLine)
	}
	return lines
}

func fitPreviewLine(s string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := make([]rune, 0, len(s))
	visibleWidth := 0
	truncated := false
	for _, r := range s {
		if r == '\t' {
			tabWidth := TAB_STOP - (visibleWidth % TAB_STOP)
			if visibleWidth+tabWidth > width {
				truncated = true
				break
			}
			for range tabWidth {
				runes = append(runes, ' ')
			}
			visibleWidth += tabWidth
			continue
		}

		charWidth := runeDisplayWidth(r)
		if charWidth > 0 && visibleWidth+charWidth > width {
			truncated = true
			break
		}
		runes = append(runes, r)
		if charWidth > 0 {
			visibleWidth += charWidth
		}
	}

	if truncated {
		if width == 1 {
			if len(runes) > 0 {
				return string(runes[:1])
			}
			return "~"
		}

		for visibleWidth >= width && len(runes) > 0 {
			lastRune := runes[len(runes)-1]
			runes = runes[:len(runes)-1]
			visibleWidth -= runeDisplayWidth(lastRune)
		}
		runes = append(runes, '~')
		visibleWidth++
	}

	if visibleWidth < width {
		runes = append(runes, []rune(strings.Repeat(" ", width-visibleWidth))...)
	}

	return string(runes)
}

func (ex *ExplorerScreen) moveToParentDirectory() {
	ex.currentDir = filepath.Dir(ex.currentDir)
}

// Initialize sets up the initial cursor position for the explorer
func (ex *ExplorerScreen) Initialize(e *Editor) {
	// Start at first file (skip header and optionally parent dir)
	ex.setCursorToFirstFile(e)
	ex.highlightSelectedFile(e)
}

// HandleKey processes key presses for the explorer screen
func (ex *ExplorerScreen) HandleKey(key int, e *Editor) (bool, bool) {
	switch key {
	case 'q', 'Q', '\x1b': // ESC or 'q' to quit
		return true, true // Close modal and restore previous state

	case ARROW_UP, ARROW_DOWN:
		ex.handleExplorerNavigation(key, e)
		ex.highlightSelectedFile(e)

	case ARROW_LEFT: // Go to parent directory
		if ex.hasParentDir {
			ex.moveToParentDirectory()
			err := ex.refreshContent()
			if err != nil {
				e.ShowError("Failed to read directory: %v", err)
				return false, false
			}
			ex.setCursorToFirstFile(e)
		}

	case '\r', ARROW_RIGHT: // Enter key
		opened := ex.openSelectedFile(e)
		if opened {
			return true, false // Close modal but keep new file state (don't restore)
		}

		// Directory was not changed, since current file has unsaved changes
		if e.dirty > 0 {
			return false, false
		}

		ex.setCursorToFirstFile(e)
	}
	ex.highlightSelectedFile(e)

	return false, false // Don't close modal
}

// handleExplorerNavigation handles arrow key navigation in the explorer
func (ex *ExplorerScreen) handleExplorerNavigation(key int, e *Editor) {
	minCy := ex.parentRowIndex()
	maxCy := len(ex.content) - 1

	switch key {
	case ARROW_UP:
		if e.cy > minCy {
			e.cy--
		}
	case ARROW_DOWN:
		if e.cy < maxCy {
			e.cy++
		}
	}
}

// highlightSelectedFile highlights the currently selected file in the explorer
func (ex *ExplorerScreen) highlightSelectedFile(e *Editor) {
	if e.cy <= 0 || e.cy >= len(ex.content) {
		return
	}

	// Reset all highlights first
	for i := 1; i < len(ex.content); i++ {
		for j := range ex.content[i].hl {
			ex.content[i].hl[j] = HL_NORMAL
		}
	}

	// Highlight current selection
	for j := range ex.content[e.cy].hl {
		ex.content[e.cy].hl[j] = HL_MATCH
	}

	// Update the editor's content reference
	e.row = ex.content
}

// openSelectedFile attempts to open the currently selected file or navigate to directory
func (ex *ExplorerScreen) openSelectedFile(e *Editor) bool {
	selectedPath, selectedEntry, hasSelection := ex.selectionAtCursor(e)
	if !hasSelection {
		return false
	}

	if selectedEntry == nil {
		ex.moveToParentDirectory()
		err := ex.refreshContent()
		if err != nil {
			e.ShowError("Failed to read directory: %v", err)
			return false
		}
		return false // Directory changed, don't close explorer
	}

	if selectedEntry.IsDir() {
		// Navigate into directory
		ex.currentDir = selectedPath
		err := ex.refreshContent()
		if err != nil {
			e.ShowError("Failed to read directory: %v", err)
			return false
		}
		return false // Directory changed, don't close explorer
	}

	if e.dirty > 0 {
		e.SetStatusMessage("File has unsaved changes")
		return false
	}

	// Open regular file
	err := e.Open(selectedPath)
	if err != nil {
		e.ShowError("Failed to open file: %v", err)
		return false
	}

	// When a file is opened from explorer we exit the modal without restoring
	// previous state, so switch mode back to normal editing explicitly.
	e.mode = EDIT_MODE

	return true // File opened successfully
}

// Explorer opens the file explorer interface using the modal system
func (e *Editor) Explorer() {
	explorerScreen := NewExplorerScreen(e, ".")
	if explorerScreen == nil {
		return // Error already shown
	}
	modalManager := NewModalManager(e, explorerScreen)
	modalManager.Show(EXPLORER_MODE)
}
