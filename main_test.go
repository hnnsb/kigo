package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyStartupPathDirectory(t *testing.T) {
	// ARRANGE
	tmpDir := t.TempDir()

	// ACT
	filePath, dirPath, err := classifyStartupPath(tmpDir)

	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filePath != "" {
		t.Fatalf("expected empty filePath, got %q", filePath)
	}

	wantDir, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve expected path: %v", err)
	}
	if filepath.Clean(dirPath) != filepath.Clean(wantDir) {
		t.Fatalf("expected dirPath %q, got %q", wantDir, dirPath)
	}
}

func TestClassifyStartupPathFile(t *testing.T) {
	// ARRANGE
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "example.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// ACT
	filePath, dirPath, err := classifyStartupPath(file)

	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dirPath != "" {
		t.Fatalf("expected empty dirPath, got %q", dirPath)
	}
	if filepath.Clean(filePath) != filepath.Clean(file) {
		t.Fatalf("expected filePath %q, got %q", file, filePath)
	}
}

func TestClassifyStartupPathMissingDefaultsToFile(t *testing.T) {
	// ARRANGE
	missing := filepath.Join(t.TempDir(), "missing.txt")

	// ACT
	filePath, dirPath, err := classifyStartupPath(missing)

	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dirPath != "" {
		t.Fatalf("expected empty dirPath, got %q", dirPath)
	}
	if filepath.Clean(filePath) != filepath.Clean(missing) {
		t.Fatalf("expected filePath %q, got %q", missing, filePath)
	}
}

func TestIsUpdateConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "single letter yes", input: "y\n", expected: true},
		{name: "word yes", input: "yes\n", expected: true},
		{name: "yes without newline", input: "yes", expected: true},
		{name: "uppercase yes", input: "YES\n", expected: true},
		{name: "no", input: "n\n", expected: false},
		{name: "empty", input: "\n", expected: false},
		{name: "whitespace", input: "   \t", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConfirmation(tt.input)
			if got != tt.expected {
				t.Fatalf("isUpdateConfirmation(%q) = %v, expected %v", tt.input, got, tt.expected)
			}
		})
	}
}
