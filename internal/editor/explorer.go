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
	explorerPinnedRows      = 2
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
		rows:      e.rows,
		totalRows: e.totalRows,
		cx:        e.cx,
		cy:        e.cy,
		colOffset: e.colOffset,
		rowOffset: e.rowOffset,
	}
}

// setEditorState restores the editor to a previously saved state
func (e *Editor) setEditorState(state EditorState) {
	e.rows = state.rows
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

func hasParentDirectory(path string) bool {
	cleanPath := filepath.Clean(path)
	return cleanPath != filepath.Dir(cleanPath)
}

func (ex *ExplorerScreen) setExplorerContent(host ModalHost) {
	host.SetRows(ex.content)
	host.SetTotalRows(len(ex.content))
	host.SetRowOffset(0)
	host.SetStatusMessage("%s", ex.GetStatusMessage())
}

func (ex *ExplorerScreen) setCursorToFirstFile(host ModalHost) {
	host.SetCy(ex.firstFileRowIndex())
	ex.setExplorerContent(host)
}

func (ex *ExplorerScreen) setCursorToFileByName(host ModalHost, fileName string) bool {
	for i, entry := range ex.files {
		if entry.Name() == fileName {
			host.SetCy(ex.rowIndexForFile(i))
			ex.setExplorerContent(host)
			return true
		}
	}
	return false
}

func (ex *ExplorerScreen) navigateToParentDirectory(host ModalHost) bool {
	if !ex.hasParentDir {
		return false
	}

	previousDirName := filepath.Base(ex.currentDir)
	ex.moveToParentDirectory()

	err := ex.refreshContent()
	if err != nil {
		host.ShowError("Failed to read directory: %v", err)
		return false
	}

	if !ex.setCursorToFileByName(host, previousDirName) {
		ex.setCursorToFirstFile(host)
	}

	return true
}

func (ex *ExplorerScreen) selectionAtCursor(host ModalHost) (selectedPath string, selectedEntry os.DirEntry, hasSelection bool) {
	return ex.selectionAtRow(host.GetCy())
}

func (ex *ExplorerScreen) selectionAtRow(cy int) (selectedPath string, selectedEntry os.DirEntry, hasSelection bool) {
	if ex.hasParentDir && cy == ex.parentRowIndex() {
		return filepath.Dir(ex.currentDir), nil, true
	}

	firstFileRow := ex.firstFileRowIndex()
	if cy < firstFileRow {
		return "", nil, false
	}

	fileIndex := cy - firstFileRow
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
	absStartDir, err := filepath.Abs(startDir)
	if err != nil {
		editor.ShowError("Failed to resolve directory path: %v", err)
		return nil
	}

	explorer := &ExplorerScreen{
		currentDir: absStartDir,
		editor:     editor,
	}
	err = explorer.refreshContent()
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
	ex.hasParentDir = hasParentDirectory(ex.currentDir)

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
	headerRow.Update(&ex.editor.Buffer)
	for i := range headerRow.hl {
		headerRow.hl[i] = HL_NORMAL
	}
	explorerRows = append(explorerRows, headerRow)
	separatorRow := DisplayLine{idx: ex.separatorRowIndex(), chars: []rune(strings.Repeat("-", ex.editor.screenCols))}
	separatorRow.Update(&ex.editor.Buffer)
	explorerRows = append(explorerRows, separatorRow)

	// Add parent directory option (unless we're at root)
	if ex.hasParentDir {
		parentText := "⏎ .."
		parentRow := DisplayLine{
			idx:   ex.parentRowIndex(),
			chars: []rune(parentText),
		}
		parentRow.Update(&ex.editor.Buffer)
		explorerRows = append(explorerRows, parentRow)
	}

	// Add files
	for i, file := range files {
		fileRow := ex.createFileDisplayRow(i, file)
		fileRow.Update(&ex.editor.Buffer)
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
func (ex *ExplorerScreen) GetSplitViewContent(rightWidth int, maxPreviewLines int, cursorRow int) ([]DisplayLine, []string) {
	// Left pane: explorer content (file list)
	leftContent := ex.GetContent()

	// Right pane: file preview
	rightPreview := ex.buildPreviewLines(rightWidth, maxPreviewLines, cursorRow)

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
func (ex *ExplorerScreen) buildPreviewLines(width int, maxLines int, cursorRow int) []string {
	if width < previewPaneMinWidth || maxLines <= 0 {
		return nil
	}
	preview := make([]string, 0, maxLines)
	selectedPath, selectedEntry, hasSelection := ex.selectionAtRow(cursorRow)

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
	if hasParentDirectory(ex.currentDir) {
		ex.currentDir = filepath.Dir(ex.currentDir)
	}
}

// Initialize sets up the initial cursor position for the explorer
func (ex *ExplorerScreen) Initialize(host ModalHost) {
	// Start at first file (skip header and optionally parent dir)
	ex.setCursorToFirstFile(host)
	ex.highlightSelectedFile(host)
}

// HandleKey processes key presses for the explorer screen
//
//	Returns (shouldCloseModal, shouldRestoreState)
func (ex *ExplorerScreen) HandleKey(key int, host ModalHost) (bool, bool) {
	switch key {
	case 'q', 'Q', '\x1b': // ESC or 'q' to quit
		return true, true // Close modal and restore previous state

	case ARROW_UP, ARROW_DOWN:
		ex.handleExplorerNavigation(key, host)
		ex.highlightSelectedFile(host)

	case ARROW_LEFT: // Go to parent directory
		ex.navigateToParentDirectory(host)

	case '\r', ARROW_RIGHT: // Enter key
		opened, directoryChanged := ex.openSelectedFile(host)
		if opened {
			return true, false // Close modal but keep new file state (don't restore)
		}

		// Directory was not changed, since current file has unsaved changes
		if host.GetDirty() > 0 {
			return false, false
		}

		if !directoryChanged {
			ex.setCursorToFirstFile(host)
		}
	}
	ex.highlightSelectedFile(host)

	return false, false // Don't close modal
}

// handleExplorerNavigation handles arrow key navigation in the explorer
func (ex *ExplorerScreen) handleExplorerNavigation(key int, host ModalHost) {
	minCy := ex.parentRowIndex()
	maxCy := len(ex.content) - 1

	switch key {
	case ARROW_UP:
		if host.GetCy() > minCy {
			host.SetCy(host.GetCy() - 1)
		}
	case ARROW_DOWN:
		if host.GetCy() < maxCy {
			host.SetCy(host.GetCy() + 1)
		}
	}
}

// highlightSelectedFile highlights the currently selected file in the explorer
func (ex *ExplorerScreen) highlightSelectedFile(host ModalHost) {
	if host.GetCy() <= 0 || host.GetCy() >= len(ex.content) {
		return
	}

	// Reset all highlights first
	for i := 1; i < len(ex.content); i++ {
		for j := range ex.content[i].hl {
			ex.content[i].hl[j] = HL_NORMAL
		}
	}

	// Highlight current selection
	for j := range ex.content[host.GetCy()].hl {
		ex.content[host.GetCy()].hl[j] = HL_MATCH
	}

	// Update the editor's content reference
	host.SetRows(ex.content)
}

// openSelectedFile attempts to open the currently selected file or navigate to directory
func (ex *ExplorerScreen) openSelectedFile(host ModalHost) (bool, bool) {
	selectedPath, selectedEntry, hasSelection := ex.selectionAtCursor(host)
	if !hasSelection {
		return false, false
	}

	if selectedEntry == nil {
		if ex.navigateToParentDirectory(host) {
			return false, true
		}
		return false, false
	}

	if selectedEntry.IsDir() {
		// Navigate into directory
		ex.currentDir = selectedPath
		err := ex.refreshContent()
		if err != nil {
			host.ShowError("Failed to read directory: %v", err)
			return false, false
		}
		ex.setCursorToFirstFile(host)
		return false, true // Directory changed, don't close explorer
	}

	if host.GetDirty() > 0 {
		host.SetStatusMessage("File has unsaved changes")
		return false, false
	}

	// Open regular file
	err := host.Open(selectedPath)
	if err != nil {
		host.ShowError("Failed to open file: %v", err)
		return false, false
	}

	// When a file is opened from explorer we exit the modal without restoring
	// previous state, so switch mode back to normal editing explicitly.
	host.SetMode(EDIT_MODE)

	return true, false // File opened successfully
}

func (e *Editor) explorerStartDir() string {
	if e.filename == "" {
		return "."
	}
	return filepath.Dir(e.filename)
}

// Explorer opens the file explorer interface using the modal system
func (e *Editor) Explorer() {
	e.ExplorerAt(e.explorerStartDir())
}

// ExplorerAt opens the file explorer at an explicit start directory.
func (e *Editor) ExplorerAt(startDir string) {
	explorerScreen := NewExplorerScreen(e, startDir)
	if explorerScreen == nil {
		return // Error already shown
	}
	modalManager := NewModalManager(e, explorerScreen)
	modalManager.Show(EXPLORER_MODE)
}
