package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestShouldShowSplitView(t *testing.T) {
	ex := &ExplorerScreen{}

	if ex.ShouldShowSplitView(minExplorerPreviewWidth - 1) {
		t.Fatalf("preview should be hidden below threshold")
	}
	if !ex.ShouldShowSplitView(minExplorerPreviewWidth) {
		t.Fatalf("preview should be visible at threshold")
	}
}

func TestFitPreviewLineWidthHandling(t *testing.T) {
	line := fitPreviewLine("abc", 5)
	if line != "abc  " {
		t.Fatalf("expected padded line, got %q", line)
	}
	if runewidth.StringWidth(line) != 5 {
		t.Fatalf("expected display width 5, got %d", runewidth.StringWidth(line))
	}

	line = fitPreviewLine("abcdef", 4)
	if runewidth.StringWidth(line) != 4 {
		t.Fatalf("expected display width 4, got %d for %q", runewidth.StringWidth(line), line)
	}

	line = fitPreviewLine("abcdef", 1)
	if line != "a" {
		t.Fatalf("expected single-character line, got %q", line)
	}

	line = fitPreviewLine("wide 中文 text", 8)
	if runewidth.StringWidth(line) != 8 {
		t.Fatalf("expected wide text to fit exactly, got width %d for %q", runewidth.StringWidth(line), line)
	}

	line = fitPreviewLine("col1\tcol2", 8)
	if strings.ContainsRune(line, '\t') {
		t.Fatalf("expected tabs to be expanded before rendering, got %q", line)
	}
	if runewidth.StringWidth(line) != 8 {
		t.Fatalf("expected tabbed text to fit exactly, got width %d for %q", runewidth.StringWidth(line), line)
	}
}

func TestBuildPreviewLinesForSelectedFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "preview.txt")
	content := "first line\nsecond line\nthird line\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	ex := &ExplorerScreen{
		currentDir: tmpDir,
		files:      entries,
	}
	e := &Editor{Viewport: Viewport{cy: 1}}

	lines := ex.buildPreviewLines(e, 24, 8)
	if len(lines) == 0 {
		t.Fatalf("expected preview lines")
	}

	for i, line := range lines {
		if runewidth.StringWidth(line) != 24 {
			t.Fatalf("line %d width mismatch: got %d", i, runewidth.StringWidth(line))
		}
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "first line") {
		t.Fatalf("expected file content in preview, got: %q", joined)
	}
}

func TestIsBinaryFileDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Test text file
	textPath := filepath.Join(tmpDir, "text.txt")
	if err := os.WriteFile(textPath, []byte("hello world\nline 2\n"), 0644); err != nil {
		t.Fatalf("failed to write text file: %v", err)
	}
	if isBinaryFile(textPath) {
		t.Fatalf("text file should not be detected as binary")
	}

	// Test binary file with null bytes
	binaryPath := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0xFF, 0xFE, 0x00}, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}
	if !isBinaryFile(binaryPath) {
		t.Fatalf("binary file with null bytes should be detected as binary")
	}

	// Test file with excessive control characters
	controlPath := filepath.Join(tmpDir, "control.bin")
	controlData := make([]byte, 100)
	for i := 0; i < 50; i++ {
		controlData[i*2] = byte(i % 8) // Low control characters
	}
	for i := 50; i < 100; i++ {
		controlData[i] = 'A' // Printable ASCII
	}
	if err := os.WriteFile(controlPath, controlData, 0644); err != nil {
		t.Fatalf("failed to write control file: %v", err)
	}
	if !isBinaryFile(controlPath) {
		t.Fatalf("file with excessive control characters should be detected as binary")
	}
}

func TestSanitizePreviewText(t *testing.T) {
	// Test removal of control characters
	input := "hello\x00world\x01test"
	result := sanitizePreviewText(input)
	if strings.Contains(result, "\x00") || strings.Contains(result, "\x01") {
		t.Fatalf("control characters should be removed, got: %q", result)
	}
	if !strings.Contains(result, "hello") || !strings.Contains(result, "world") {
		t.Fatalf("text content should be preserved, got: %q", result)
	}

	// Test preservation of Unicode
	input = "café résumé 中文"
	result = sanitizePreviewText(input)
	if !strings.Contains(result, "café") || !strings.Contains(result, "中文") {
		t.Fatalf("Unicode should be preserved, got: %q", result)
	}

	// Test tab and whitespace preservation
	input = "hello\tworld  test"
	result = sanitizePreviewText(input)
	if !strings.Contains(result, "\t") {
		t.Fatalf("tabs should be preserved, got: %q", result)
	}

	// Test replacement of control chars with bullet
	input = "hello\x02world"
	result = sanitizePreviewText(input)
	if !strings.Contains(result, "•") {
		t.Fatalf("control characters should be replaced with bullet, got: %q", result)
	}
}
