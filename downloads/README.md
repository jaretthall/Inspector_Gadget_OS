# Inspector Gadget OS Downloads Directory

This directory contains downloaded development tools and dependencies organized for the Inspector Gadget OS project.

## Current Downloads

### Core Development Tools
- `../go1.22.5.linux-amd64.tar.gz` - Go 1.22.5 for Linux
- `../go1.23.4.linux-amd64.tar.gz` - Go 1.23.4 for Linux (newer version)
- `../get-docker.sh` - Docker installation script

### GPU Development
- `../cuda-keyring_1.1-1_all.deb` - NVIDIA CUDA toolkit keyring
- `../amdgpu-install_6.2.60202-1_all.deb` - AMD GPU driver installer

## Setup Instructions

### For Linux/WSL2:
```bash
chmod +x ../setup-dev-env.sh
./setup-dev-env.sh
```

### For Windows:
```powershell
# Run as Administrator
../setup-dev-env.ps1
```

## Additional Downloads Needed

### Kali Linux Tools (Phase 4: Week 7-8)
- [ ] Kali Linux ISO for tool extraction
- [ ] Metasploit framework
- [ ] Nmap source code
- [ ] Burp Suite Community

### Alpine Linux (Phase 3: Week 5-6)
- [ ] Alpine Linux minimal ISO
- [ ] Alpine Package Keeper (apk) tools
- [ ] Alpine SDK for custom packages

### MCP Servers (Phase 2: Week 3-4)
- [ ] MCP filesystem server container
- [ ] MCP git server container
- [ ] MCP web search server container

### Build Tools
- [ ] QEMU for cross-platform building
- [ ] Buildah for container creation
- [ ] Ansible for configuration management

## Directory Structure

```
downloads/
├── go/                     # Go language downloads
├── docker/                 # Docker and container tools
├── gpu/                    # GPU drivers and toolkits
├── kali/                   # Kali Linux tools and packages
├── alpine/                 # Alpine Linux base and packages
├── mcp/                    # Model Context Protocol servers
├── build-tools/            # OS building and packaging tools
└── third-party/            # External dependencies
```

## Usage

All downloaded tools should be verified with checksums and signatures before installation. The setup scripts will handle installation and configuration automatically.

## Next Phase Downloads

As you progress through the roadmap phases, additional downloads will be organized in this directory structure to keep the development environment clean and manageable.