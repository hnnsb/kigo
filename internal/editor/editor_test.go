package editor

import (
	"testing"
)

func TestEditorRowDeleteChar(t *testing.T) {
	e := &Editor{}
	// Create a test row
	row := &DisplayLine{
		idx:           0,
		chars:         []rune("hello"),
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Initialize the render and hl slices
	row.Update(&e.Buffer)

	// Test deleting a character
	row.deleteChar(1, &e.Buffer) // Delete 'e' from "hello"

	// Check if the character was deleted correctly
	expected := "hllo"
	actual := string(row.chars)

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}

	if len(row.chars) != 4 {
		t.Errorf("Expected chars slice length 4, got %d", len(row.chars))
	}
}

func TestEditorRowDeleteCharMultiple(t *testing.T) {
	e := &Editor{}
	// Create a test row
	row := &DisplayLine{
		idx:           0,
		chars:         []rune("abc"),
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Initialize the render and hl slices
	row.Update(&e.Buffer)

	// Test deleting multiple characters
	row.deleteChar(0, &e.Buffer) // Delete 'a' from "abc" -> "bc"
	row.deleteChar(0, &e.Buffer) // Delete 'b' from "bc" -> "c"

	// Check if the characters were deleted correctly
	expected := "c"
	actual := string(row.chars)

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}

	if len(row.chars) != 1 {
		t.Errorf("Expected chars slice length 1, got %d", len(row.chars))
	}
}
