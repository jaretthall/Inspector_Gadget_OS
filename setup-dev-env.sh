#!/bin/bash
# Inspector Gadget OS Development Environment Setup Script
# Phase 1: Foundation Setup (Weeks 1-2) - Task 2

set -e

echo "🤖 Inspector Gadget OS Development Environment Setup"
echo "======================================================"

# Check if running on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script is designed for Linux. Please run on Ubuntu 22.04+ or similar."
    exit 1
fi

# Update package lists
echo "📦 Updating package lists..."
sudo apt-get update

# Install essential build tools
echo "🔧 Installing core development tools..."
sudo apt-get install -y \
    build-essential \
    git \
    make \
    cmake \
    gcc \
    clang \
    curl \
    wget \
    unzip \
    ca-certificates \
    gnupg \
    lsb-release

# Install Go if not already installed
if ! command -v go &> /dev/null; then
    echo "🐹 Installing Go 1.22.5..."
    if [ -f "./go1.22.5.linux-amd64.tar.gz" ]; then
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf ./go1.22.5.linux-amd64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin
    else
        echo "❌ Go tarball not found. Please download go1.22.5.linux-amd64.tar.gz"
        exit 1
    fi
else
    echo "✅ Go already installed: $(go version)"
fi

# Install Docker if not already installed
if ! command -v docker &> /dev/null; then
    echo "🐳 Installing Docker..."
    if [ -f "./get-docker.sh" ]; then
        chmod +x ./get-docker.sh
        ./get-docker.sh
        sudo usermod -aG docker $USER
    else
        curl -fsSL https://get.docker.com -o get-docker.sh
        chmod +x get-docker.sh
        ./get-docker.sh
        sudo usermod -aG docker $USER
    fi
else
    echo "✅ Docker already installed: $(docker --version)"
fi

# Install Node.js for web UI development
if ! command -v node &> /dev/null; then
    echo "📱 Installing Node.js 18..."
    curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
    sudo apt-get install -y nodejs
else
    echo "✅ Node.js already installed: $(node --version)"
fi

# Install NVIDIA CUDA toolkit (if NVIDIA GPU detected)
if lspci | grep -i nvidia > /dev/null; then
    echo "🎮 NVIDIA GPU detected. Installing CUDA toolkit..."
    if [ -f "./cuda-keyring_1.1-1_all.deb" ]; then
        sudo dpkg -i ./cuda-keyring_1.1-1_all.deb
        sudo apt-get update
        sudo apt-get -y install cuda-toolkit-12-2
    else
        wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb
        sudo dpkg -i cuda-keyring_1.1-1_all.deb
        sudo apt-get update
        sudo apt-get -y install cuda-toolkit-12-2
    fi
    echo 'export PATH=/usr/local/cuda/bin:$PATH' >> ~/.bashrc
    echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH' >> ~/.bashrc
fi

# Install AMD ROCm (if AMD GPU detected)
if lspci | grep -i amd | grep -i vga > /dev/null; then
    echo "🔴 AMD GPU detected. Installing ROCm..."
    if [ -f "./amdgpu-install_6.2.60202-1_all.deb" ]; then
        sudo dpkg -i ./amdgpu-install_6.2.60202-1_all.deb
        sudo apt-get update
        sudo apt-get install -y rocm-dkms rocm-libs rocm-dev
    else
        wget https://repo.radeon.com/amdgpu-install/6.2.6/ubuntu/jammy/amdgpu-install_6.2.60202-1_all.deb
        sudo dpkg -i amdgpu-install_6.2.60202-1_all.deb
        sudo apt-get update
        sudo apt-get install -y rocm-dkms rocm-libs rocm-dev
    fi
fi

# Install additional development dependencies
echo "📚 Installing additional development dependencies..."
sudo apt-get install -y \
    sqlite3 \
    libsqlite3-dev \
    pkg-config \
    libssl-dev \
    python3 \
    python3-pip \
    alpine-sdk \
    qemu-user-static \
    binfmt-support

# Set up Go workspace
echo "🏗️  Setting up Go workspace..."
export GO111MODULE=on
cd /tmp && go mod download github.com/casbin/casbin/v2
cd -

# Verify installations
echo ""
echo "✅ Development Environment Verification:"
echo "======================================"
echo "Go:      $(go version 2>/dev/null || echo 'Not installed')"
echo "Docker:  $(docker --version 2>/dev/null || echo 'Not installed')"
echo "Node.js: $(node --version 2>/dev/null || echo 'Not installed')"
echo "Git:     $(git --version 2>/dev/null || echo 'Not installed')"
echo "Make:    $(make --version 2>/dev/null | head -1 || echo 'Not installed')"
echo "CMake:   $(cmake --version 2>/dev/null | head -1 || echo 'Not installed')"

if command -v nvidia-smi &> /dev/null; then
    echo "CUDA:    $(nvidia-smi --version | grep 'CUDA Version' || echo 'Not available')"
fi

echo ""
echo "🎉 Development environment setup complete!"
echo "Please run 'source ~/.bashrc' or restart your terminal to apply PATH changes."
echo ""
echo "Next steps:"
echo "1. Create GitHub organization 'inspector-gadget-os'"
echo "2. Set up CI/CD pipeline with GitHub Actions"
echo "3. Begin O-LLaMA development (Phase 2)"