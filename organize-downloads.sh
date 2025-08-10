#!/bin/bash
# Organize Inspector Gadget OS Downloads
# Moves downloaded files to proper directory structure

echo "📁 Organizing Inspector Gadget OS Downloads..."

# Create downloads directory structure
mkdir -p downloads/{go,docker,gpu,kali,alpine,mcp,build-tools,third-party}

# Move Go downloads
if [ -f "go1.22.5.linux-amd64.tar.gz" ]; then
    mv go1.22.5.linux-amd64.tar.gz downloads/go/
    echo "✅ Moved Go 1.22.5 to downloads/go/"
fi

if [ -f "go1.23.4.linux-amd64.tar.gz" ]; then
    mv go1.23.4.linux-amd64.tar.gz downloads/go/
    echo "✅ Moved Go 1.23.4 to downloads/go/"
fi

# Move Docker downloads
if [ -f "get-docker.sh" ]; then
    mv get-docker.sh downloads/docker/
    echo "✅ Moved Docker installer to downloads/docker/"
fi

# Move GPU downloads
if [ -f "cuda-keyring_1.1-1_all.deb" ]; then
    mv cuda-keyring_1.1-1_all.deb downloads/gpu/
    echo "✅ Moved CUDA keyring to downloads/gpu/"
fi

if [ -f "amdgpu-install_6.2.60202-1_all.deb" ]; then
    mv amdgpu-install_6.2.60202-1_all.deb downloads/gpu/
    echo "✅ Moved AMD GPU installer to downloads/gpu/"
fi

# Update setup scripts to use new paths
echo "🔧 Updating setup scripts to use organized download paths..."

# Update Linux setup script
sed -i 's|./go1.22.5.linux-amd64.tar.gz|downloads/go/go1.22.5.linux-amd64.tar.gz|g' setup-dev-env.sh
sed -i 's|./get-docker.sh|downloads/docker/get-docker.sh|g' setup-dev-env.sh
sed -i 's|./cuda-keyring_1.1-1_all.deb|downloads/gpu/cuda-keyring_1.1-1_all.deb|g' setup-dev-env.sh
sed -i 's|./amdgpu-install_6.2.60202-1_all.deb|downloads/gpu/amdgpu-install_6.2.60202-1_all.deb|g' setup-dev-env.sh

echo ""
echo "📂 Download Directory Structure:"
echo "downloads/"
find downloads -type f -exec echo "├── {}" \;

echo ""
echo "✅ Downloads organized successfully!"
echo "Use ./setup-dev-env.sh (Linux) or ./setup-dev-env.ps1 (Windows) to install."