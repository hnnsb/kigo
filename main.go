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
		err = editor.Open(args[0])
		if err != nil {
			editor.ShowError("%v", err)
		}
	}

	for {
		editor.RefreshScreen()
		editor.ProcessKeypress()
	}
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
