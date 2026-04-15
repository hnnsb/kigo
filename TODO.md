# KIGO - Development Roadmap

## Known Issues & Bug Fixes

### General Bugs

- No critical bugs reported currently

## Feature Improvements

- [ ] Config via config file (user settings, keybindings, themes)
- [ ] Beautify explorer screen
  - [x] Sort directories first, then files
  - [x] Add file type indicators/icons
  - [x] Show file sizes
  - [ ] Add gitignore-style hidden file toggle
  - [ ] Improve preview metadata (encoding, line count, binary hints)
- [ ] Search/Find functionality enhancements
- [ ] Syntax highlighting for more languages

## Go Idiomatic Refactoring Opportunities

### ✅ Completed (v1.0.0)

**1. Error Handling & Recovery**

- [x] Replace `die()` function with proper Go error handling
- [x] Return errors from functions instead of fatal exits
- [x] Create custom error types
- [x] Implement graceful error recovery

**2. Global State Management**

- [x] Eliminate global variable `E`
- [x] Create proper Editor struct with dependency injection
- [x] Pass editor context through function calls

**3. Constants and Naming Conventions**

- [x] Group related constants using typed constants and iota
- [x] Create enum-like types for keys, colors, and styles

**11. String and Byte Handling**

- [x] Use strings.Builder for concatenation
- [x] Implement proper Unicode support with runes
- [x] Consistent string/byte usage

**13. Rendering and View Model Refactor**

- [x] Introduce `Buffer` model for file/content state
- [x] Introduce `Viewport` model for cursor and scroll state
- [x] Extract terminal drawing to `ScreenRenderer` in `renderer.go`
- [x] Move scroll calculations into `Viewport.Scroll()`
- [x] Route editor draw calls through renderer abstraction
- [x] Add split-view renderer support for explorer preview

### 🔄 In Progress / Next Steps

**4. Function Organization and Interfaces** (Medium Priority)

- [ ] Define narrower interfaces for terminal I/O and file I/O boundaries
- [ ] Continue moving content-centric logic from `Editor` to `Buffer` methods
- [ ] Consider splitting into subpackages (terminal, renderer, syntax) if needed
- [ ] Follow Go naming conventions for methods (methods on receivers)

**5. Memory Management and Slices** (Medium Priority)

- [ ] Simplify complex slice operations with Go idioms
- [ ] Use proper slice initialization patterns
- [ ] Review manual buffer management patterns
- [ ] Remove manual length tracking where possible

**6. Type System Improvements** (Medium Priority)

- [ ] Consider typed constants for keys instead of raw `int`
- [ ] Implement enum-like types for syntax highlighting rules
- [ ] Add helper methods to `DisplayLine` and `editorSyntax` types
- [ ] Review pointer vs value receiver patterns

**14. Refactor Validation and Tests** (Medium Priority)

- [ ] Add focused unit tests for `Viewport.Scroll()` edge cases
- [ ] Add tests for renderer clipping with wide Unicode runes
- [ ] Add tests for split-view rendering fallbacks at narrow terminal widths
- [ ] Add modal integration tests covering save/restore state transitions

### 📋 Future Work (Lower Priority)

**8. Configuration and Initialization** (Lower Priority)

- [ ] Create configuration file system (YAML/TOML)
- [ ] Use builder pattern for complex initialization
- [ ] Add user configuration support
- [ ] Configuration validation

**9. File I/O and Resource Management** (Lower Priority)

- [ ] Optimize file operations for large files
- [ ] Ensure proper resource cleanup with defer
- [ ] Add context support for cancellable operations
- [ ] Consider using embed for built-in templates

**10. Testing and Testability** (Lower Priority)

- [ ] Enhance unit tests in editor_test.go
- [ ] Implement table-driven tests
- [ ] Create test helpers for common scenarios
- [ ] Add integration tests for editor workflows

**12. Syntax Highlighting Architecture** (Lower Priority)

- [ ] Design extensible syntax highlighting system
- [ ] Implement registry pattern for language definitions
- [ ] Add more language support (Python, Rust, JavaScript, etc.)
- [ ] Consider streaming approach for very large files

**7. Concurrency and Channels** (Future Exploration)

- [ ] Non-blocking input handling with goroutines
- [ ] Background file saving operations
- [ ] Proper signal handling
- [ ] Background syntax highlighting

## Implementation Strategy

1. **Current Focus**: Stabilize Buffer/Viewport/Renderer refactor with tests
2. **Next Phase**: Continue extracting responsibilities from `Editor` (sections 4, 5, 6)
3. **Then**: Configuration system and testing improvements (sections 8, 10, 14)
4. **Future**: Concurrency features and advanced syntax architecture (sections 7, 12)

# Issues

- [x] Make it possible to open Directories and start in Explorer View
  - [x] when explorer is closed without selecting a file, create a new one at diretcory?
- Line Count in status bar changes when opening Explorer
- [x] Add Line numbers to editor (toggleable)
- allow combination of different text stylings instead of being limited to two
- [/] add more cursor navigation features
  - selecting text (shift+arrows)
    - select word (ctrl+shift+arrows)
  - copy (ctrl+c), paste already works?!
  - [x] cursor movement by word (ctrl+arrows)
  - [x] word delete (ctrl+del/ctrl+backspace)
  - [x] delete line
- [/] add function to tab key?
  - [x] Basic typing
  - indention (tab/shift+tab)
  - navigation through parentheses
- [x] preserve indent on line break
- auto close parantheses (in code)
- undo/redo (ctrl+y, ctrl+z)
- fix opening the same file twice
  - different changes just overwrite each other
  - add updating or warning that new changes are overwritten
- add multi cursor support
  - to create cursors: (ctrl+alt+arrows)
- [x] recreate shell history after close?
  - currently it is erased and just the last editor windows can be scrolled to
- open multiple files and allow switching between them
  - show tabs in status bar?
- add code navigation tools, jump from symbol declaration to usage and vice versa
- fuzzy search
- diff viewer: show difference between changes and file
- sort by dirs and files in preview
- [x] remove tilde in empty explorer lines "~"
- git support, at least highlighting in file?
- Save Position in file when opening/closing files
- toggle ins-mode?
- Find/Replace Mode
- Parenthese Highlighting to show in which parenthese the cursor is.
- [x] scrolling in explorer hides header whenn list exceeds height, and header cannot reappear.
