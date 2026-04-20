package ansi

import "testing"

func TestEscGraphic(t *testing.T) {
	expected := "\x1b[30m"
	actual := string(EscGraphic(COLOR_BLACK))
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}
