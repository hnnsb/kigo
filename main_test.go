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
