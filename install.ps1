$ErrorActionPreference = "Stop"

$Repo = "yourname/kigo"
$Binary = "kigo"
$InstallDir = "$env:LOCALAPPDATA\Programs\kigo"

Write-Host "🚀 Installing $Binary..." -ForegroundColor Cyan

# Detect architecture
$arch = $env:PROCESSOR_ARCHITECTURE
switch ($arch) {
    "AMD64" { $arch = "amd64" }
    "ARM64" { $arch = "arm64" }
    default {
        Write-Error "❌ Unsupported architecture: $arch"
        exit 1
    }
}

$os = "windows"

$File = "$Binary-$os-$arch.zip"
$Url = "https://github.com/$Repo/releases/latest/download/$File"

# Create install directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$TempFile = Join-Path $env:TEMP $File
$TempDir = Join-Path $env:TEMP "kigo-install"

Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

Write-Host "⬇️ Downloading $File..."
Invoke-WebRequest -Uri $Url -OutFile $TempFile

Write-Host "📦 Extracting..."
Expand-Archive -Path $TempFile -DestinationPath $TempDir -Force

$ExePath = Join-Path $TempDir "$Binary.exe"

if (!(Test-Path $ExePath)) {
    Write-Error "❌ Binary not found after extraction"
    exit 1
}

Write-Host "📥 Installing..."
Copy-Item $ExePath "$InstallDir\$Binary.exe" -Force

# Add to PATH (user-level)
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Write-Host "🔧 Added to PATH (restart terminal required)"
}

Write-Host "✅ Installed successfully!"
Write-Host "👉 Run: kigo --help"

Remove-Item $TempFile -Force
Remove-Item -Recurse -Force $TempDir