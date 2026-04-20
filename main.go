package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hnnsb/kigo/internal/editor"
	"github.com/hnnsb/kigo/internal/version"
)

func main() {
	args := os.Args[1:]

	if len(args) >= 1 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-h", "--help":
			printHelp()
			return
		case "-v", "--version":
			printVersion()
			return
		case "--update":
			if !confirmUpdate() {
				fmt.Println("Update canceled.")
				return
			}
			if err := update(); err != nil {
				fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	logFile, err := getLogFile("kigo")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	editor := editor.NewEditor(logger)
	err = editor.EnableRawMode()
	if err != nil {
		editor.Die("enabling raw mode: %s", err.Error())
	}
	defer editor.RestoreTerminal()

	err = editor.Init()
	if err != nil {
		editor.Die("initializing editor: %s", err.Error())
	}

	editor.Debug("Kigo-%s initialized successfully", version.Version)
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

func printVersion() {
	fmt.Printf("KIGO editor version %s (commit %s, built at %s)\n", version.Version, version.Commit, version.Date)
}

func printHelp() {
	helpLines := []string{
		"Usage: kigo [file_or_directory]",
		"",
		"A simple terminal-based text editor written in Go.",
		"",
		"Options:",
		"  -h, --help       Show this help message",
		"  -v, --version    Show version information",
		"  --update         Check for updates and install the latest version",
		"",
		"Examples:",
		"  kigo                # Open editor with no file",
		"  kigo myfile.txt     # Open or create myfile.txt",
		"  kigo mydirectory/   # Open file explorer at mydirectory/"}
	fmt.Print(strings.Join(helpLines, "\n"))
}

func update() error {
	fmt.Println("Checking for updates...")

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "iwr https://raw.githubusercontent.com/hnnsb/kigo/main/install.ps1 -UseBasicParsing | iex")
	} else {
		cmd = exec.Command("sh", "-c", "curl -sL https://raw.githubusercontent.com/hnnsb/kigo/main/install.sh | bash")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running update script: %w", err)
	}

	return nil
}

func confirmUpdate() bool {
	fmt.Print("This will download and run the latest install script. Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}

	return isUpdateConfirmation(response)
}

func isUpdateConfirmation(response string) bool {
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes":
		return true
	default:
		return false
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
