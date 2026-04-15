# KIGO

go implementation of a basic text editor. It follows the tutorial of the kilo editor implemented in C
(https://viewsourcecode.org/snaptoken/kilo/, https://antirez.com/news/108)

This is a learning project to get familiar with go.

## Installation

You can install kigo on Linux, macOS, and Windows without needing Go installed.

### Linux / macOS

Install with a single command:

```bash
curl -sL https://raw.githubusercontent.com/hnnsb/kigo/main/install.sh | bash
```

This will:

- detect your operating system and architecture
- download the latest release
- install kigo to /usr/local/bin

### 🪟 Windows (PowerShell)

Run in PowerShell:

```PowerShell
iwr https://raw.githubusercontent.com/hnnsb/kigo/main/install.ps1 -UseBasicParsing | iex
```

This will:

- download the latest release for Windows
- install kigo.exe to your user programs directory
- add it to your PATH (may require restarting the terminal)

## Updating

### Option 1: Re-run the installer (recommended)

To update to the latest version at any time, simply run the install command again:

#### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/hnnsb/kigo/main/install.sh | bash
```

#### Windows

```PowerShell
iwr https://raw.githubusercontent.com/hnnsb/kigo/main/install.ps1 -UseBasicParsing | iex
```

This will overwrite the existing installation with the latest release.

### Option 2: Built-in update command

kigo also includes a built-in update command:

```
kigo --update
```

This will:

- check for the latest available version
- download and install the newest release
- replace the current binary automatically

## Manual installation (optional)

You can also download binaries directly from the GitHub Releases page:

https://github.com/hnnsb/kigo/releases

After downloading:

1. extract the archive
2. move the binary into your PATH

## Debugging

Live read log file

```PowerShell
Get-Content .\debug.log -Wait
```

```Bash
tail debug.log -f
```
