# Inspector Gadget OS Development Guide

Welcome to the Inspector Gadget OS development environment! This guide will help you set up your development environment and contribute to the ultimate AI Swiss Army knife.

## 🚀 Quick Start

### Prerequisites
- Linux (Ubuntu 22.04+ recommended) or WSL2 on Windows
- 16GB+ RAM (32GB recommended for full development)
- NVIDIA GPU (recommended) or AMD GPU with ROCm support
- 100GB+ free disk space

### 1. Clone the Repository
```bash
git clone https://github.com/your-org/inspector-gadget-os.git
cd inspector-gadget-os
```

### 2. Set Up Development Environment

**Linux/WSL2:**
```bash
chmod +x setup-dev-env.sh
./setup-dev-env.sh
```

**Windows (PowerShell as Administrator):**
```powershell
./setup-dev-env.ps1
```

### 3. Organize Downloads
```bash
chmod +x organize-downloads.sh
./organize-downloads.sh
```

### 4. Build and Test
```bash
# Build all components
make build

# Run tests
make test

# Start development environment
make dev
```

## 📁 Project Structure

```
inspector-gadget-os/
├── o-llama/                    # Enhanced Ollama fork with file access
│   ├── internal/auth/          # JWT authentication
│   ├── internal/rbac/          # Role-based access control
│   ├── internal/safefs/        # Secure file system access
│   └── internal/mcp/           # Model Context Protocol integration
├── gadget-framework/           # Plugin architecture for gadgets
│   ├── gadget/                 # Core gadget interfaces
│   ├── command/                # "Go Go Gadget" command parsing
│   └── cmd/go-go-gadget/       # CLI tool for gadget management
├── gadgets/                    # Individual gadget implementations
│   ├── network-scanner/        # AI-enhanced network reconnaissance
│   ├── vulnerability-assessment/ # Smart vulnerability analysis
│   └── ultron/                 # Personal assistant integration
├── os-core/                    # Base operating system
│   ├── configs/                # System configurations
│   └── system/                 # OS-level components
├── web-ui/                     # React-based management interface
├── downloads/                  # Organized development tools
└── .github/workflows/          # CI/CD pipeline
```

## 🛠️ Development Workflow

### Phase Structure
Development follows the roadmap phases:
- **Phase 1** (Weeks 1-2): Foundation Setup ✅
- **Phase 2** (Weeks 3-4): O-LLaMA Development
- **Phase 3** (Weeks 5-6): Base OS Development  
- **Phase 4** (Weeks 7-8): Kali Integration
- **Phase 5** (Weeks 9-10): Ultron Integration
- **Phase 6** (Weeks 11-12): Gadget Framework
- **Phase 7** (Weeks 13-14): System Integration
- **Phase 8** (Weeks 15-16): USB Boot System

### Current Phase: Phase 2 - O-LLaMA Development

**Next Steps:**
1. Fork Ollama repository to `inspector-gadget-os/o-llama`
2. Add file system access with security controls
3. Implement JWT authentication and RBAC
4. Add MCP integration for tool execution
5. Create comprehensive test suite

### Working with Components

**O-LLaMA Enhanced Runtime:**
```bash
cd o-llama
go mod tidy
go test ./...
go run ./cmd/integrated-server
```

**Gadget Framework:**
```bash
cd gadget-framework  
go build ./cmd/go-go-gadget
./go-go-gadget list
./go-go-gadget install network-scanner
```

**Web UI Development:**
```bash
cd web-ui
npm install
npm run dev  # Development server at http://localhost:5173
npm run build
```

## 🧪 Testing

### Unit Tests
```bash
# Test specific component
cd o-llama && go test ./...
cd gadget-framework && go test ./...

# Test all Go components
make test-go
```

### Integration Tests
```bash
# End-to-end testing
make test-integration

# Security testing
make test-security
```

### Web UI Tests
```bash
cd web-ui
npm run test
npm run test:e2e
```

## 🔒 Security Guidelines

### File System Access
- All file operations go through `safefs` package
- Path validation prevents directory traversal
- Size limits prevent resource exhaustion
- Audit logging for all file operations

### Authentication
- JWT tokens with configurable expiry
- RBAC with Casbin for fine-grained permissions
- API keys for external integrations
- Secure defaults (localhost-only binding)

### Container Security
- Minimal attack surface with Alpine Linux base
- Non-root user execution
- Resource limits and namespacing
- Security scanning in CI/CD pipeline

## 🎯 Gadget Development

### Creating a New Gadget

1. **Generate scaffold:**
```bash
./go-go-gadget create my-awesome-gadget
```

2. **Implement interface:**
```go
type MyGadget struct{}

func (g *MyGadget) Name() string { return "My Awesome Gadget" }
func (g *MyGadget) Category() gadget.GadgetCategory { return gadget.CategoryProductivity }
func (g *MyGadget) Execute(ctx context.Context, args []string) (*gadget.Result, error) {
    // Your gadget logic here
}
```

3. **Test and package:**
```bash
go test ./...
./go-go-gadget package my-awesome-gadget
```

### Gadget Guidelines
- Implement security best practices
- Include AI integration where appropriate
- Provide clear documentation and examples
- Follow "Go Go Gadget" command patterns
- Respect user privacy and data ownership

## 📊 Monitoring and Observability

### Metrics
```bash
# View system metrics
curl http://localhost:9090/metrics

# Gadget performance
curl http://localhost:8080/api/gadgets/metrics
```

### Logging
```bash
# View system logs
journalctl -u inspector-gadget-os

# Component logs
docker logs inspector-gadget-os-ollama
```

## 🚢 Deployment

### Development Environment
```bash
docker-compose up -d
```

### Building OS Image
```bash
./scripts/build-os.sh appliance amd64
```

### Creating USB Installer
```bash
./scripts/create-installer.sh inspector-gadget-os-appliance-amd64.img
```

## 🤝 Contributing

### Before You Start
1. Read [CONTRIBUTING.md](./CONTRIBUTING.md)
2. Check [GitHub Issues](https://github.com/your-org/inspector-gadget-os/issues)
3. Join our [Discord Community](https://discord.gg/inspector-gadget)

### Development Process
1. Create feature branch: `git checkout -b feature/amazing-feature`
2. Make changes following style guidelines
3. Add tests for new functionality
4. Run full test suite: `make test-all`
5. Create pull request with detailed description

### Code Style
- Go: Follow `go fmt` and `go vet`
- TypeScript: ESLint + Prettier configuration
- Commit messages: [Conventional Commits](https://conventionalcommits.org/)

## 📚 Resources

### Documentation
- [Architecture Overview](./docs/architecture.md)
- [API Reference](./docs/api.md)
- [Gadget SDK](./docs/gadget-sdk.md)
- [Security Guide](./docs/security.md)

### Community
- [GitHub Discussions](https://github.com/your-org/inspector-gadget-os/discussions)
- [Discord Server](https://discord.gg/inspector-gadget)
- [Reddit Community](https://reddit.com/r/InspectorGadgetOS)

### External Resources
- [Ollama Documentation](https://ollama.ai/docs)
- [Model Context Protocol](https://modelcontextprotocol.io)
- [Alpine Linux Guide](https://wiki.alpinelinux.org)

## 🐛 Troubleshooting

### Common Issues

**GPU not detected:**
```bash
nvidia-smi  # Check NVIDIA GPU
rocm-smi    # Check AMD GPU
```

**Port conflicts:**
```bash
# Check what's using port 8080
netstat -tulpn | grep 8080
```

**Permission errors:**
```bash
# Ensure user is in docker group
sudo usermod -aG docker $USER
# Logout and login again
```

**Build failures:**
```bash
# Clean build cache
make clean
docker system prune -a
```

### Getting Help
1. Check [Troubleshooting Guide](./docs/troubleshooting.md)
2. Search [GitHub Issues](https://github.com/your-org/inspector-gadget-os/issues)
3. Ask on [Discord](https://discord.gg/inspector-gadget)
4. Create detailed bug report with logs

---

**Ready to build the future of AI-powered operating systems?** 🤖🚀

Start with Phase 2: O-LLaMA Development and let's bring Inspector Gadget OS to life!