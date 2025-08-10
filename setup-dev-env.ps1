# Inspector Gadget OS Development Environment Setup Script (Windows)
# Phase 1: Foundation Setup (Weeks 1-2) - Task 2

Write-Host "🤖 Inspector Gadget OS Development Environment Setup (Windows)" -ForegroundColor Green
Write-Host "=================================================================" -ForegroundColor Green

# Check if running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "❌ This script requires Administrator privileges. Please run as Administrator." -ForegroundColor Red
    exit 1
}

# Install Chocolatey if not present
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Host "📦 Installing Chocolatey package manager..."
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
}

# Install core development tools
Write-Host "🔧 Installing core development tools..."
choco install -y git
choco install -y make
choco install -y mingw
choco install -y cmake
choco install -y curl
choco install -y wget

# Install Go if not already installed
if (!(Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "🐹 Installing Go..."
    if (Test-Path ".\go1.22.5.linux-amd64.tar.gz") {
        # Note: On Windows, we should use the Windows version
        Write-Host "ℹ️  Found Linux Go tarball. For Windows, please download the Windows version from https://golang.org/dl/"
        choco install -y golang
    } else {
        choco install -y golang
    }
} else {
    Write-Host "✅ Go already installed: $(go version)"
}

# Install Docker Desktop
if (!(Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "🐳 Installing Docker Desktop..."
    choco install -y docker-desktop
} else {
    Write-Host "✅ Docker already installed: $(docker --version)"
}

# Install Node.js for web UI development
if (!(Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "📱 Installing Node.js..."
    choco install -y nodejs
} else {
    Write-Host "✅ Node.js already installed: $(node --version)"
}

# Install WSL2 for Linux development
Write-Host "🐧 Setting up WSL2 for Linux development..."
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform

# Install Ubuntu on WSL2
wsl --install -d Ubuntu-22.04

# Install Visual Studio Code
Write-Host "📝 Installing Visual Studio Code..."
choco install -y vscode

# Install Git extensions
Write-Host "📋 Installing Git extensions..."
choco install -y git-lfs
choco install -y github-desktop

# Install CUDA toolkit (if NVIDIA GPU detected)
$gpu = Get-WmiObject -Class Win32_VideoController | Where-Object {$_.Name -like "*NVIDIA*"}
if ($gpu) {
    Write-Host "🎮 NVIDIA GPU detected. Installing CUDA toolkit..."
    choco install -y cuda
}

# Set up environment variables
Write-Host "🔧 Setting up environment variables..."
[Environment]::SetEnvironmentVariable("GO111MODULE", "on", "User")

# Create development directories
Write-Host "📁 Creating development directories..."
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\go"
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\go\src"
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\go\bin"
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\go\pkg"

# Verify installations
Write-Host ""
Write-Host "✅ Development Environment Verification:" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green

try { Write-Host "Go:      $(go version)" } catch { Write-Host "Go:      Not installed" }
try { Write-Host "Docker:  $(docker --version)" } catch { Write-Host "Docker:  Not installed" }
try { Write-Host "Node.js: $(node --version)" } catch { Write-Host "Node.js: Not installed" }
try { Write-Host "Git:     $(git --version)" } catch { Write-Host "Git:     Not installed" }
try { Write-Host "Make:    $(make --version | Select-Object -First 1)" } catch { Write-Host "Make:    Not installed" }

if (Get-Command nvcc -ErrorAction SilentlyContinue) {
    Write-Host "CUDA:    $(nvcc --version | Select-String 'release')"
}

Write-Host ""
Write-Host "🎉 Development environment setup complete!" -ForegroundColor Green
Write-Host "Please restart your terminal or PowerShell to apply changes." -ForegroundColor Yellow
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Restart PowerShell/Terminal" -ForegroundColor White
Write-Host "2. Create GitHub organization 'inspector-gadget-os'" -ForegroundColor White
Write-Host "3. Set up CI/CD pipeline with GitHub Actions" -ForegroundColor White
Write-Host "4. Begin O-LLaMA development (Phase 2)" -ForegroundColor White
Write-Host ""
Write-Host "For Linux development, use WSL2:" -ForegroundColor Cyan
Write-Host "wsl" -ForegroundColor White
Write-Host "cd /mnt/d/Inspector_Gadget_OS" -ForegroundColor White
Write-Host "./setup-dev-env.sh" -ForegroundColor White