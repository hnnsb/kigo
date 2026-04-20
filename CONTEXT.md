# KIGO - Terminal Text Editor Context for LLM Agents

## Project Overview

KIGO is a terminal-based text editor written in Go, inspired by the Kilo editor tutorial. It is a learning project focused on building a practical editor with Unicode support, syntax highlighting, file exploration, and modal workflows.

## Technical Stack

- Language: Go 1.24.2
- Dependencies:
  - golang.org/x/term v0.33.0 (terminal/raw mode handling)
  - golang.org/x/sys v0.34.0 (system calls)
  - github.com/mattn/go-runewidth v0.0.16 (Unicode display width)
- Repository: github.com/hnnsb/kigo

## Project Structure

```
kigo/
+-- main.go                 # Program entry point and main event loop
+-- go.mod                  # Go module definition
+-- go.sum                  # Dependency checksums
+-- README.md               # User-facing project documentation
+-- CONTEXT.md              # LLM context and architecture overview
+-- TODO.md                 # Roadmap and refactor tracking
+-- internal/
    +-- editor/
    |   +-- editor.go       # Core models + editing/input logic
    |   +-- renderer.go     # ScreenRenderer drawing pipeline
    |   +-- modal.go        # Modal interfaces and modal manager
    |   +-- explorer.go     # Explorer modal and preview content
    |   +-- help.go         # Help modal
    |   +-- ansi.go         # ANSI constants and terminal helpers
    |   +-- editor_test.go  # Editor tests
    |   +-- explorer_test.go# Explorer tests
    +-- version/
        +-- version.go      # Build-time version metadata
```

## Core Architecture

### 1. Editor as Coordinator

The Editor type coordinates the application rather than implementing every concern itself.

- Embeds Buffer (content state) and Viewport (cursor/scroll state)
- Handles input dispatch, mode transitions, and high-level workflow
- Delegates all terminal drawing to ScreenRenderer
- Maintains modal context via activeModal

### 2. Buffer Model

Buffer represents file/content state:

- row []DisplayLine
- totalRows
- dirty
- filename
- syntax

This keeps file-backed content concerns separate from terminal viewport concerns.

### 3. Viewport Model

Viewport tracks where and how content is shown:

- Cursor: cx, cy
- Render cursor x: rx
- Scroll offsets: rowOffset, colOffset
- Visible area: screenRows, screenCols

Viewport.Scroll(totalRows, rows) computes cursor-relative scrolling.

### 4. ScreenRenderer

ScreenRenderer in internal/editor/renderer.go owns terminal output rendering.

- Draws body rows, status bar, message bar, and cursor position
- Applies syntax styles/colors
- Handles Unicode width-aware clipping and alignment via go-runewidth
- Supports split-view rendering for modals that implement SplitViewModal

### 5. Modal System

modal.go defines an interface-based modal architecture:

- ModalScreen for regular modal behavior
- SplitViewModal for left-pane content + right-pane preview
- ModalManager for save/restore of editor state and modal loop control

### 6. Explorer Integration

ExplorerScreen (internal/editor/explorer.go):

- Reads and sorts directory entries (directories first)
- Produces DisplayLine content for modal list rendering
- Provides optional split-view preview content for files
- Uses modal save/restore state transitions when entering/exiting explorer mode

## Key Data Structures

```go
type Editor struct {
    Viewport
    Buffer
    statusMessage     string
    statusMessageTime time.Time
    mode              int
    terminal          *Terminal
    renderer          *ScreenRenderer
    activeModal       ModalScreen
}

type Buffer struct {
    totalRows int
    row       []DisplayLine
    dirty     int
    filename  string
    syntax    *editorSyntax
}

type Viewport struct {
    cx, cy     int
    rx         int
    rowOffset  int
    colOffset  int
    screenRows int
    screenCols int
}

type DisplayLine struct {
    idx           int
    chars         []rune
    render        []rune
    hl            []int
    hlOpenComment bool
}

type ModalScreen interface {
    GetContent() []DisplayLine
    GetTitle() string
    GetStatusMessage() string
    HandleKey(key int, e *Editor) (bool, bool)
    Initialize(e *Editor)
}

type SplitViewModal interface {
    ModalScreen
    ShouldShowSplitView(screenCols int) bool
    GetSplitViewContent(e *Editor, rightWidth int, maxPreviewLines int) ([]DisplayLine, []string)
}
```

## Current Status

### Completed

- Global state removed in favor of Editor struct
- Unicode support implemented with rune-based text handling
- Buffer and Viewport model split implemented
- ScreenRenderer extracted from core editor logic
- Scroll behavior moved into Viewport.Scroll
- Split-view modal rendering integrated for explorer preview

### Active Refactor Direction

- Continue extracting content-centric methods from Editor into Buffer-focused behavior
- Expand tests around viewport edge cases and renderer clipping
- Tighten interface boundaries for terminal I/O and file I/O

## Key Features

- File open/save/new
- Cursor/navigation editing controls
- Search/find workflow
- Syntax highlighting
- Modal explorer/help flows
- Unicode-aware rendering and editing

## Runtime Flow (High Level)

1. main.go creates Editor and initializes terminal + screen.
2. Main loop repeatedly calls RefreshScreen() and ProcessKeypress().
3. Editor delegates rendering to ScreenRenderer.
4. Modal workflows run through ModalManager and restore state on exit.

## Architecture Decisions

1. Keep Editor as orchestration layer.
2. Keep content and viewport concerns split (Buffer vs Viewport).
3. Keep terminal painting isolated in ScreenRenderer.
4. Keep modal behavior interface-driven with optional split-view extension.
5. Prefer rune-based pipelines end-to-end for Unicode correctness.
