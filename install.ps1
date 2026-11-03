# BramandScript Windows PowerShell Installer
# Usage: irm https://raw.githubusercontent.com/Codeleafly/bramandScript/master/install.ps1 | iex

$ErrorActionPreference = 'Stop'
$Repo = "Codeleafly/bramandScript"

Write-Host "✦ [INFO] Detecting Windows Architecture..." -ForegroundColor Cyan

$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

$InstallDir = "$HOME\.bramand\bin"
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Host "✦ [INFO] Fetching latest release from GitHub ($Repo)..." -ForegroundColor Cyan

try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Tag = $Release.tag_name
} catch {
    $Tag = "v0.1.1"
}

$ArchiveName = "bramand-windows-$Arch-$Tag.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$ArchiveName"

$TempZip = "$env:TEMP\$ArchiveName"
Write-Host "✦ [INFO] Downloading $DownloadUrl..." -ForegroundColor Cyan

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip
    Expand-Archive -Path $TempZip -DestinationPath $InstallDir -Force
    Remove-Item -Path $TempZip -Force

    # Ensure alias copy
    Copy-Item -Path "$InstallDir\bramand.exe" -Destination "$InstallDir\bmd.exe" -Force
} catch {
    Write-Warning "Could not download pre-compiled binary. Checking local Go installation..."
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "✦ [INFO] Compiling native Windows binary using Go..." -ForegroundColor Cyan
        go build -ldflags="-s -w" -o "$InstallDir\bramand.exe" main.go
        Copy-Item -Path "$InstallDir\bramand.exe" -Destination "$InstallDir\bmd.exe" -Force
    } else {
        Write-Error "Failed to install BramandScript. Please install Go or download binaries from https://github.com/$Repo/releases"
    }
}

# Add to User PATH if not present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$UserPath", "User")
    $env:Path = "$InstallDir;$env:Path"
}

Write-Host "✔ [OK] BramandScript successfully installed to $InstallDir!" -ForegroundColor Green
Write-Host "       Commands: 'bramand', 'bmd'" -ForegroundColor Green

& "$InstallDir\bmd.exe" version
