# O-LLaMA Enhancements

## Overview
O-LLaMA is an enhanced fork of Ollama that adds:
- File system access with security boundaries
- Authentication and RBAC (Role-Based Access Control)
- MCP (Model Context Protocol) integration
- Audit logging for all operations

## Key Additions

### 1. SafeFS Package
Location: `internal/safefs/`
- Path validation to prevent directory traversal
- File size limits
- Extension filtering
- Audit logging

### 2. Authentication Layer
Location: `internal/auth/`
- JWT token authentication
- User management
- Session handling

### 3. RBAC System
Location: `internal/rbac/`
- Casbin-based permission management
- Fine-grained access control
- Role definitions

### 4. MCP Integration
Location: `internal/mcp/`
- MCP server discovery
- Tool execution framework
- Server lifecycle management

## Building O-LLaMA

```bash
# Install dependencies
go mod tidy

# Build the enhanced server
go build -o o-llama ./cmd/o-llama

# Run with enhanced features
./o-llama serve --enable-fs --enable-auth
```

## Configuration

See `configs/runtime.yaml` for configuration options.