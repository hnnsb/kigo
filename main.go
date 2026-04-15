package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hnnsb/kigo/editor"
)

func main() {
	logFile, err := getLogFile("kigo")
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	editor := editor.NewEditor(logger)
	args := os.Args[1:]
	err = editor.EnableRawMode()
	if err != nil {
		editor.Die("enabling raw mode: %s", err.Error())
	}
	defer editor.RestoreTerminal()

	err = editor.Init()
	if err != nil {
		editor.Die("initializing editor: %s", err.Error())
	}

	editor.Debug("Editor initialized successfully")
	editor.SetStatusMessage("HELP: Ctrl-S = save | Ctrl-Q = quit | Ctrl-F = find")

	if len(args) >= 1 {
		editor.Debug("Processing startup argument: %q", args[0])
		startupFile, startupDir, startupErr := classifyStartupPath(args[0])
		if startupErr != nil {
			editor.ShowError("%v", startupErr)

		} else if startupDir != "" {
			editor.Debug("Opening directory: %q", startupDir)
			editor.ExplorerAt(startupDir)

		} else if startupFile != "" {
			editor.Debug("Opening file: %q", startupFile)
			err = editor.Open(startupFile)
			if err != nil {
				editor.ShowError("%v", err)
			}
		}
	}

	for {
		editor.RefreshScreen()
		editor.ProcessKeypress()
	}
}

func classifyStartupPath(arg string) (filePath string, dirPath string, err error) {
	cleanArg := filepath.Clean(arg)
	info, statErr := os.Stat(cleanArg)
	if statErr == nil {
		if info.IsDir() {
			absDir, absErr := filepath.Abs(cleanArg)
			if absErr != nil {
				return "", "", fmt.Errorf("failed to resolve directory path: %w", absErr)
			}
			return "", absDir, nil
		}
		return cleanArg, "", nil
	}

	if os.IsNotExist(statErr) {
		// Missing path is treated as a file target, matching existing behavior.
		return cleanArg, "", nil
	}

	return "", "", fmt.Errorf("failed to access path %q: %w", cleanArg, statErr)
}

func getLogFile(appName string) (*os.File, error) {
	if os.Getenv("DEBUG") == "1" {
		file, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			fmt.Fprintf(file, "--- %s ---\n", time.Now().Format("2006-01-02_15:04:05"))
		}
		return file, err
	}

	baseDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	logDir := filepath.Join(baseDir, appName, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(logDir, "app.log")

	return os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
}
