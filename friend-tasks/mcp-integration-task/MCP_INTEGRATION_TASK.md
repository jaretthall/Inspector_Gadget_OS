# MCP Integration Task - Inspector Gadget OS

## 🎯 Task Overview

**Phase**: Phase 2 - O-LLaMA Development (Week 4)  
**Component**: Model Context Protocol Integration  
**Difficulty**: Intermediate  
**Estimated Time**: 4-6 hours  

## 📋 Task Description

Implement Model Context Protocol (MCP) integration to enable the enhanced Ollama server to execute tools and access external resources securely. This will allow AI models to interact with file systems, git repositories, and other external services through standardized MCP servers.

## 🎯 Success Criteria

- [ ] MCP client can discover and connect to MCP servers
- [ ] Tool execution with proper permission validation
- [ ] Resource management (start/stop/monitor MCP servers)
- [ ] Comprehensive error handling and logging
- [ ] Integration tests with filesystem and git MCP servers
- [ ] Security controls prevent unauthorized tool access

## 📁 Files to Create/Modify

### Core MCP Implementation
```
o-llama/internal/mcp/
├── client.go          # MCP client implementation
├── manager.go         # MCP server lifecycle management  
├── protocol.go        # MCP protocol message handling
├── transport.go       # Communication transport layer
├── types.go           # MCP protocol types and structures
├── security.go        # Security validation for MCP operations
└── README.md          # MCP integration documentation
```

### Integration Points
```
o-llama/cmd/integrated-server/main.go  # Initialize MCP manager
o-llama/internal/integration/mcp.go    # MCP-Ollama bridge
o-llama/configs/mcp-servers.yaml      # MCP server configurations
```

### Tests
```
o-llama/internal/mcp/
├── client_test.go
├── manager_test.go
├── protocol_test.go
└── integration_test.go
```

## 🔧 Implementation Details

### 1. MCP Protocol Types (types.go)
```go
package mcp

import (
    "context"
    "time"
)

// MCP protocol message types
type MessageType string

const (
    MessageTypeRequest     MessageType = "request"
    MessageTypeResponse    MessageType = "response" 
    MessageTypeNotification MessageType = "notification"
)

// Core MCP message structure
type Message struct {
    ID      string      `json:"id,omitempty"`
    Type    MessageType `json:"type"`
    Method  string      `json:"method,omitempty"`
    Params  interface{} `json:"params,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *Error      `json:"error,omitempty"`
}

// MCP error structure
type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// Tool definition from MCP server
type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
}

// Tool execution request
type ToolCall struct {
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}

// Tool execution result
type ToolResult struct {
    Content []ContentItem `json:"content"`
    IsError bool          `json:"isError"`
}

type ContentItem struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    Data []byte `json:"data,omitempty"`
}

// Resource definition
type Resource struct {
    URI         string `json:"uri"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    MimeType    string `json:"mimeType,omitempty"`
}
```

### 2. MCP Client (client.go)
```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
    
    "go.uber.org/zap"
)

type Client struct {
    logger    *zap.Logger
    transport Transport
    tools     map[string]Tool
    resources map[string]Resource
    mu        sync.RWMutex
    
    // Request tracking
    pendingRequests map[string]chan Message
    nextID          int64
}

func NewClient(transport Transport, logger *zap.Logger) *Client {
    return &Client{
        logger:          logger,
        transport:       transport,
        tools:           make(map[string]Tool),
        resources:       make(map[string]Resource),
        pendingRequests: make(map[string]chan Message),
    }
}

func (c *Client) Connect(ctx context.Context) error {
    c.logger.Info("Connecting to MCP server")
    
    if err := c.transport.Connect(ctx); err != nil {
        c.logger.Error("Failed to connect to MCP server", zap.Error(err))
        return fmt.Errorf("transport connection failed: %w", err)
    }
    
    // Start message handler
    go c.handleMessages()
    
    // Initialize MCP session
    if err := c.initialize(ctx); err != nil {
        c.logger.Error("MCP initialization failed", zap.Error(err))
        return fmt.Errorf("MCP initialization failed: %w", err)
    }
    
    c.logger.Info("Successfully connected to MCP server")
    return nil
}

func (c *Client) initialize(ctx context.Context) error {
    // Send initialize request
    initMsg := Message{
        Type:   MessageTypeRequest,
        Method: "initialize",
        Params: map[string]interface{}{
            "protocolVersion": "2024-11-05",
            "capabilities": map[string]interface{}{
                "tools":     map[string]interface{}{},
                "resources": map[string]interface{}{},
            },
            "clientInfo": map[string]interface{}{
                "name":    "Inspector Gadget OS",
                "version": "0.1.3",
            },
        },
    }
    
    response, err := c.sendRequest(ctx, initMsg)
    if err != nil {
        return fmt.Errorf("initialize request failed: %w", err)
    }
    
    if response.Error != nil {
        return fmt.Errorf("initialize error: %s", response.Error.Message)
    }
    
    // List available tools
    if err := c.refreshTools(ctx); err != nil {
        c.logger.Warn("Failed to refresh tools", zap.Error(err))
    }
    
    // List available resources  
    if err := c.refreshResources(ctx); err != nil {
        c.logger.Warn("Failed to refresh resources", zap.Error(err))
    }
    
    return nil
}

func (c *Client) ExecuteTool(ctx context.Context, toolCall ToolCall) (*ToolResult, error) {
    c.logger.Info("Executing MCP tool",
        zap.String("tool_name", toolCall.Name),
        zap.Any("arguments", toolCall.Arguments))
    
    // Validate tool exists
    c.mu.RLock()
    tool, exists := c.tools[toolCall.Name]
    c.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("tool %s not found", toolCall.Name)
    }
    
    // Validate arguments against schema
    if err := c.validateArguments(toolCall.Arguments, tool.InputSchema); err != nil {
        return nil, fmt.Errorf("argument validation failed: %w", err)
    }
    
    // Execute tool
    msg := Message{
        Type:   MessageTypeRequest,
        Method: "tools/call",
        Params: map[string]interface{}{
            "name":      toolCall.Name,
            "arguments": toolCall.Arguments,
        },
    }
    
    response, err := c.sendRequest(ctx, msg)
    if err != nil {
        return nil, fmt.Errorf("tool execution failed: %w", err)
    }
    
    if response.Error != nil {
        return &ToolResult{
            Content: []ContentItem{{
                Type: "text",
                Text: fmt.Sprintf("Tool error: %s", response.Error.Message),
            }},
            IsError: true,
        }, nil
    }
    
    // Parse result
    var result ToolResult
    if err := json.Unmarshal(response.Result.([]byte), &result); err != nil {
        return nil, fmt.Errorf("failed to parse tool result: %w", err)
    }
    
    c.logger.Info("Tool execution completed",
        zap.String("tool_name", toolCall.Name),
        zap.Bool("is_error", result.IsError))
    
    return &result, nil
}

// Additional methods: sendRequest, handleMessages, refreshTools, etc.
```

### 3. MCP Manager (manager.go)
```go
package mcp

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "go.uber.org/zap"
)

type ServerConfig struct {
    Name        string                 `yaml:"name"`
    Type        string                 `yaml:"type"`        // "stdio", "sse", "websocket"
    Command     []string               `yaml:"command"`     // For stdio servers
    URL         string                 `yaml:"url"`         // For network servers
    Args        []string               `yaml:"args"`
    Environment map[string]string      `yaml:"environment"`
    Enabled     bool                   `yaml:"enabled"`
    Timeout     time.Duration          `yaml:"timeout"`
}

type Manager struct {
    logger  *zap.Logger
    servers map[string]*ServerInstance
    configs []ServerConfig
    mu      sync.RWMutex
}

type ServerInstance struct {
    Config ServerConfig
    Client *Client
    Status ServerStatus
    
    // Process management for stdio servers
    process    *os.Process
    stdin      io.WriteCloser
    stdout     io.ReadCloser
    stderr     io.ReadCloser
    startTime  time.Time
    lastPing   time.Time
}

type ServerStatus string

const (
    StatusStopped   ServerStatus = "stopped"
    StatusStarting  ServerStatus = "starting"
    StatusRunning   ServerStatus = "running"
    StatusError     ServerStatus = "error"
)

func NewManager(configs []ServerConfig, logger *zap.Logger) *Manager {
    return &Manager{
        logger:  logger,
        servers: make(map[string]*ServerInstance),
        configs: configs,
    }
}

func (m *Manager) StartAll(ctx context.Context) error {
    m.logger.Info("Starting all MCP servers")
    
    var wg sync.WaitGroup
    errors := make(chan error, len(m.configs))
    
    for _, config := range m.configs {
        if !config.Enabled {
            continue
        }
        
        wg.Add(1)
        go func(cfg ServerConfig) {
            defer wg.Done()
            if err := m.StartServer(ctx, cfg.Name); err != nil {
                errors <- fmt.Errorf("failed to start %s: %w", cfg.Name, err)
            }
        }(config)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    var startupErrors []error
    for err := range errors {
        startupErrors = append(startupErrors, err)
    }
    
    if len(startupErrors) > 0 {
        return fmt.Errorf("failed to start %d servers: %v", len(startupErrors), startupErrors)
    }
    
    m.logger.Info("All MCP servers started successfully")
    return nil
}

func (m *Manager) StartServer(ctx context.Context, name string) error {
    // Find config
    var config ServerConfig
    found := false
    for _, cfg := range m.configs {
        if cfg.Name == name {
            config = cfg
            found = true
            break
        }
    }
    
    if !found {
        return fmt.Errorf("server %s not found in configuration", name)
    }
    
    m.logger.Info("Starting MCP server",
        zap.String("name", name),
        zap.String("type", config.Type))
    
    // Create server instance
    instance := &ServerInstance{
        Config:    config,
        Status:    StatusStarting,
        startTime: time.Now(),
    }
    
    // Create transport based on type
    var transport Transport
    var err error
    
    switch config.Type {
    case "stdio":
        transport, err = NewStdioTransport(config.Command, config.Args, config.Environment)
    case "websocket":
        transport, err = NewWebSocketTransport(config.URL)
    default:
        return fmt.Errorf("unsupported transport type: %s", config.Type)
    }
    
    if err != nil {
        return fmt.Errorf("failed to create transport: %w", err)
    }
    
    // Create MCP client
    client := NewClient(transport, m.logger.With(zap.String("mcp_server", name)))
    instance.Client = client
    
    // Connect to server
    if err := client.Connect(ctx); err != nil {
        instance.Status = StatusError
        return fmt.Errorf("failed to connect: %w", err)
    }
    
    instance.Status = StatusRunning
    instance.lastPing = time.Now()
    
    // Store instance
    m.mu.Lock()
    m.servers[name] = instance
    m.mu.Unlock()
    
    // Start health monitoring
    go m.monitorServer(instance)
    
    m.logger.Info("MCP server started successfully", zap.String("name", name))
    return nil
}

// Additional methods: StopServer, GetServer, monitorServer, etc.
```

### 4. Configuration (mcp-servers.yaml)
```yaml
# MCP Server Configurations for Inspector Gadget OS
mcp:
  enabled: true
  servers:
    - name: filesystem
      type: stdio
      command: ["npx", "@modelcontextprotocol/server-filesystem"]
      args: ["/home/user/documents", "/mnt/external"]
      enabled: true
      timeout: 30s
      environment:
        MCP_LOG_LEVEL: "info"
      
    - name: git
      type: stdio  
      command: ["npx", "@modelcontextprotocol/server-git"]
      args: ["/home/user/projects"]
      enabled: true
      timeout: 30s
      environment:
        GIT_CONFIG_GLOBAL: "/etc/gitconfig"
        
    - name: brave-search
      type: stdio
      command: ["npx", "@modelcontextprotocol/server-brave-search"]
      enabled: false  # Requires API key
      timeout: 60s
      environment:
        BRAVE_API_KEY: "${BRAVE_API_KEY}"
        
    - name: postgres
      type: stdio
      command: ["npx", "@modelcontextprotocol/server-postgres"]
      args: ["postgresql://user:pass@localhost/inspector_gadget"]
      enabled: false
      timeout: 30s
```

### 5. Integration Tests (integration_test.go)
```go
package mcp

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap/zaptest"
)

func TestMCPFilesystemIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    logger := zaptest.NewLogger(t)
    
    // Create temporary directory for testing
    tempDir := t.TempDir()
    testFile := filepath.Join(tempDir, "test.txt")
    err := os.WriteFile(testFile, []byte("Hello, MCP!"), 0644)
    require.NoError(t, err)
    
    // Create filesystem MCP server config
    config := ServerConfig{
        Name:    "test-filesystem",
        Type:    "stdio",
        Command: []string{"npx", "@modelcontextprotocol/server-filesystem"},
        Args:    []string{tempDir},
        Enabled: true,
        Timeout: 30 * time.Second,
    }
    
    // Start MCP manager
    manager := NewManager([]ServerConfig{config}, logger)
    
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    err = manager.StartServer(ctx, "test-filesystem")
    require.NoError(t, err)
    defer manager.StopAll(ctx)
    
    // Get client
    client := manager.GetServer("test-filesystem").Client
    require.NotNil(t, client)
    
    // Test file read
    result, err := client.ExecuteTool(ctx, ToolCall{
        Name: "read_file",
        Arguments: map[string]interface{}{
            "path": "test.txt",
        },
    })
    
    require.NoError(t, err)
    assert.False(t, result.IsError)
    assert.Len(t, result.Content, 1)
    assert.Equal(t, "text", result.Content[0].Type)
    assert.Equal(t, "Hello, MCP!", result.Content[0].Text)
}

func TestMCPGitIntegration(t *testing.T) {
    // Similar test for git MCP server
    // Test git status, git log, etc.
}

func TestMCPSecurityValidation(t *testing.T) {
    logger := zaptest.NewLogger(t)
    
    // Test that MCP operations respect security boundaries
    tempDir := t.TempDir()
    
    config := ServerConfig{
        Name:    "restricted-filesystem",
        Type:    "stdio", 
        Command: []string{"npx", "@modelcontextprotocol/server-filesystem"},
        Args:    []string{tempDir}, // Only allow access to temp dir
        Enabled: true,
        Timeout: 30 * time.Second,
    }
    
    manager := NewManager([]ServerConfig{config}, logger)
    
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    err := manager.StartServer(ctx, "restricted-filesystem")
    require.NoError(t, err)
    defer manager.StopAll(ctx)
    
    client := manager.GetServer("restricted-filesystem").Client
    
    // Test path traversal prevention
    result, err := client.ExecuteTool(ctx, ToolCall{
        Name: "read_file",
        Arguments: map[string]interface{}{
            "path": "../../../etc/passwd", // Should be blocked
        },
    })
    
    // Should either error or return empty/denied result
    if err == nil {
        assert.True(t, result.IsError, "Path traversal should be blocked")
    }
}
```

## 🔒 Security Considerations

1. **Tool Validation**: Validate all tool calls against schemas
2. **Path Restrictions**: Ensure MCP servers respect file system boundaries
3. **Resource Limits**: Prevent resource exhaustion from MCP operations
4. **Authentication**: Ensure MCP tools respect user permissions
5. **Audit Logging**: Log all MCP tool executions for security review

## 🧪 Testing Requirements

- Unit tests for all major components
- Integration tests with real MCP servers
- Security tests for path traversal and permission bypass
- Performance tests for concurrent tool execution
- Error handling tests for network failures

## 📚 Resources

- [Model Context Protocol Specification](https://modelcontextprotocol.io/docs)
- [MCP Server Implementations](https://github.com/modelcontextprotocol)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)

## 🎯 Deliverables

1. Complete MCP client implementation with all core methods
2. MCP server lifecycle manager with health monitoring  
3. Configuration system for MCP servers
4. Security validation layer for tool execution
5. Comprehensive test suite with >80% coverage
6. Integration with existing logging and monitoring
7. Documentation and usage examples

This task will enable the AI models in Inspector Gadget OS to safely interact with external tools and resources, forming a crucial foundation for the gadget framework!