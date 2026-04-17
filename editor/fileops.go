package editor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (b *Buffer) RowsToString() ([]byte, int) {
	var buf strings.Builder
	lineEnding := getLineEnding()

	// Pre-calculate total size for efficiency
	totalSize := 0
	for _, row := range b.rows {
		totalSize += len(row.chars) + len(lineEnding) // +len(lineEnding) for line ending
	}
	buf.Grow(totalSize)

	for _, row := range b.rows {
		buf.WriteString(string(row.chars))
		buf.WriteString(lineEnding)
	}

	result := buf.String()
	return []byte(result), len(result)
}

func (e *Editor) RowsToString() ([]byte, int) {
	return e.Buffer.RowsToString()
}

func (b *Buffer) Open(filename string) error {
	b.filename = filename
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", filename, err)
	}
	defer file.Close()

	// Reset buffer state, because we are opening a new file
	b.rows = make([]DisplayLine, 0)
	b.totalRows = 0
	b.SelectSyntaxHighlight()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing newlines and carriage returns
		for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
			line = line[:len(line)-1]
		}

		runes := []rune(line)
		b.InsertRow(b.totalRows, runes, len(runes))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading file %q: %w", filename, err)
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
	return e.Buffer.Open(filename)
}

func (b *Buffer) SaveToFile() (int, error) {
	buf, length := b.RowsToString()

	file, err := os.OpenFile(b.filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("opening file %q for write: %w", b.filename, err)
	}
	defer file.Close()

	err = file.Truncate(int64(length))
	if err != nil {
		return 0, fmt.Errorf("truncating file %q: %w", b.filename, err)
	}

	bytesWritten, err := file.Write(buf)
	if err != nil {
		return 0, fmt.Errorf("writing file %q: %w", b.filename, err)
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
		e.Buffer.SelectSyntaxHighlight()
	}

	length, err := e.Buffer.SaveToFile()
	if err != nil {
		e.SetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	e.SetStatusMessage("%d bytes written to disk", length)
}
