# KIGO - Development Roadmap

## Known Issues & Bug Fixes

### Current (fix/unicode-support branch)

- [ ] Verify Unicode character width calculations in modal content
- [ ] Test cursor positioning with multi-byte characters (emoji, CJK)
- [ ] Ensure file explorer displays Unicode filenames correctly

### General Bugs

- No critical bugs reported currently

## Feature Improvements

- [ ] Config via config file (user settings, keybindings, themes)
- [ ] Beautify explorer screen
  - [ ] Sort directories first, then files
  - [ ] Add file type icons or indicators
  - [ ] Show file sizes
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

### 🔄 In Progress / Next Steps

**4. Function Organization and Interfaces** (Medium Priority)

- [ ] Define interfaces for terminal operations, file operations, rendering
- [ ] Convert package-level functions to methods on appropriate structs
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
- [ ] Add helper methods to `editorRow` and `editorSyntax` types
- [ ] Review pointer vs value receiver patterns

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

1. **Current Focus**: Verify unicode-support branch completion
2. **Next Phase**: Function organization and type system improvements (sections 4, 6)
3. **Then**: Configuration system and testing improvements (sections 8, 10)
4. **Future**: Concurrency features and advanced architecture (sections 7, 12)
