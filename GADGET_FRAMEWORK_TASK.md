# Gadget Framework Plugin Architecture - Detailed Specification

## 📋 Task Overview

Implement the core plugin system for Inspector Gadget OS that enables dynamic loading, management, and execution of "gadgets" (plugins/tools). This system will serve as the foundation for the "Go Go Gadget [Anything]" philosophy.

## 🎯 Goals

1. **Modular Architecture**: Each tool/capability is a separate gadget
2. **Dynamic Loading**: Load/unload gadgets at runtime without restart
3. **Natural Language Interface**: Parse "Go Go Gadget [Tool]" commands
4. **Security**: Isolate gadgets with proper permission boundaries
5. **Extensibility**: Easy framework for creating new gadgets

## 📁 File Structure

```
gadget-framework/
├── gadget/
│   ├── interface.go           # Core gadget interface definition
│   ├── manager.go            # Gadget lifecycle manager
│   ├── types.go              # Common types, enums, errors
│   ├── config.go             # Configuration structures
│   └── registry/
│       ├── registry.go       # Gadget registry and discovery
│       └── registry_test.go  # Registry tests
├── command/
│   ├── parser.go             # "Go Go Gadget" command parser
│   ├── parser_test.go        # Parser tests
│   └── context.go            # Command execution context
├── examples/
│   ├── weather_integration.go # Weather gadget integration
│   └── hello_gadget.go       # Simple example gadget
├── internal/
│   ├── loader/
│   │   ├── loader.go         # Plugin loading mechanisms
│   │   └── loader_test.go    # Loader tests
│   └── security/
│       ├── permissions.go    # Permission system
│       └── sandbox.go        # Gadget isolation
└── cmd/
    └── gadget-cli/
        └── main.go           # CLI for testing gadgets
```

## 🔧 Core Components

### 1. Gadget Interface (`gadget/interface.go`)

```go
// Gadget represents a plugin/tool in the Inspector Gadget OS
type Gadget interface {
    // Metadata
    Name() string
    Description() string
    Category() GadgetCategory
    Version() string
    Author() string
    
    // Lifecycle
    Initialize(ctx context.Context, config Config) error
    Start() error
    Stop() error
    Health() HealthStatus
    
    // Execution
    Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)
    
    // AI Integration (for future O-LLaMA integration)
    GetAIPrompts() []AIPrompt
    ProcessAIResponse(response string) (*Action, error)
    
    // Dependencies and Permissions
    RequiredPermissions() []Permission
    Dependencies() []Dependency
    Capabilities() []Capability
}

// GadgetCategory defines the type of gadget
type GadgetCategory string

const (
    CategoryAI           GadgetCategory = "ai"
    CategorySecurity     GadgetCategory = "security"
    CategoryProductivity GadgetCategory = "productivity"
    CategorySystem       GadgetCategory = "system"
    CategoryCustom       GadgetCategory = "custom"
    CategoryDevelopment  GadgetCategory = "development"
)

// HealthStatus represents gadget health
type HealthStatus struct {
    Status  string            `json:"status"`  // "healthy", "degraded", "unhealthy"
    Message string            `json:"message"`
    Details map[string]string `json:"details"`
}

// ExecuteRequest contains command execution parameters
type ExecuteRequest struct {
    Command     string            `json:"command"`
    Args        []string          `json:"args"`
    Flags       map[string]string `json:"flags"`
    Context     ExecutionContext  `json:"context"`
    UserID      string            `json:"user_id"`
    Permissions []Permission      `json:"permissions"`
}

// ExecuteResponse contains command execution results
type ExecuteResponse struct {
    Success    bool              `json:"success"`
    Output     string            `json:"output"`
    ErrorMsg   string            `json:"error_msg,omitempty"`
    Data       interface{}       `json:"data,omitempty"`
    Actions    []Action          `json:"actions,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
    ExitCode   int               `json:"exit_code"`
}
```

### 2. Gadget Manager (`gadget/manager.go`)

```go
// GadgetManager handles the lifecycle of all gadgets
type GadgetManager struct {
    gadgets     map[string]Gadget
    registry    *registry.Registry
    config      *Config
    permissions *security.PermissionManager
    logger      Logger
    mutex       sync.RWMutex
}

// Key Methods to Implement:
func NewGadgetManager(config *Config) *GadgetManager
func (gm *GadgetManager) LoadGadget(path string) error
func (gm *GadgetManager) UnloadGadget(name string) error
func (gm *GadgetManager) GetGadget(name string) (Gadget, error)
func (gm *GadgetManager) ListGadgets() []GadgetInfo
func (gm *GadgetManager) ExecuteCommand(cmd *ParsedCommand) (*ExecuteResponse, error)
func (gm *GadgetManager) Start() error
func (gm *GadgetManager) Stop() error
func (gm *GadgetManager) HealthCheck() map[string]HealthStatus

// GadgetInfo contains metadata about a loaded gadget
type GadgetInfo struct {
    Name         string        `json:"name"`
    Description  string        `json:"description"`
    Category     GadgetCategory `json:"category"`
    Version      string        `json:"version"`
    Author       string        `json:"author"`
    Status       string        `json:"status"`
    LoadedAt     time.Time     `json:"loaded_at"`
    LastExecuted time.Time     `json:"last_executed,omitempty"`
}
```

### 3. Command Parser (`command/parser.go`)

```go
// CommandParser parses "Go Go Gadget" natural language commands
type CommandParser struct {
    gadgetManager *gadget.Manager
    aiEnabled     bool
    logger        Logger
}

// ParsedCommand represents a parsed "Go Go Gadget" command
type ParsedCommand struct {
    Original    string            `json:"original"`
    Trigger     string            `json:"trigger"`     // "Go Go Gadget"
    Tool        string            `json:"tool"`        // "Weather"
    Action      string            `json:"action"`      // "Get", "Check", etc.
    Target      string            `json:"target"`      // "New York", "192.168.1.1"
    Args        []string          `json:"args"`
    Flags       map[string]string `json:"flags"`
    Context     ExecutionContext  `json:"context"`
    Confidence  float64           `json:"confidence"`  // 0.0-1.0
}

// Key Methods to Implement:
func NewCommandParser(gm *gadget.Manager) *CommandParser
func (cp *CommandParser) Parse(input string) (*ParsedCommand, error)
func (cp *CommandParser) ValidateCommand(cmd *ParsedCommand) error
func (cp *CommandParser) GetSuggestions(partial string) []string
func (cp *CommandParser) ExecuteCommand(cmd *ParsedCommand, userID string) (*ExecuteResponse, error)

// Example parsing patterns:
// "Go Go Gadget Weather New York" -> tool="weather", action="get", target="New York"
// "Go Go Gadget Network Scan 192.168.1.0/24" -> tool="network-scanner", action="scan", target="192.168.1.0/24"
// "Go Go Gadget Project Status" -> tool="ultron", action="status"
```

### 4. Registry System (`gadget/registry/registry.go`)

```go
// Registry manages gadget discovery and metadata
type Registry struct {
    gadgets      map[string]*GadgetMetadata
    searchIndex  map[string][]string // keyword -> gadget names
    categories   map[GadgetCategory][]string
    mutex        sync.RWMutex
    manifestPath string
}

// GadgetMetadata contains comprehensive gadget information
type GadgetMetadata struct {
    Name         string            `json:"name" yaml:"name"`
    Description  string            `json:"description" yaml:"description"`
    Category     GadgetCategory    `json:"category" yaml:"category"`
    Version      string            `json:"version" yaml:"version"`
    Author       string            `json:"author" yaml:"author"`
    License      string            `json:"license" yaml:"license"`
    Homepage     string            `json:"homepage" yaml:"homepage"`
    Repository   string            `json:"repository" yaml:"repository"`
    Keywords     []string          `json:"keywords" yaml:"keywords"`
    Dependencies []Dependency      `json:"dependencies" yaml:"dependencies"`
    Permissions  []Permission      `json:"permissions" yaml:"permissions"`
    Config       map[string]string `json:"config" yaml:"config"`
    Binary       string            `json:"binary" yaml:"binary"`
    EntryPoint   string            `json:"entry_point" yaml:"entry_point"`
    InstallCmd   string            `json:"install_cmd" yaml:"install_cmd"`
}

// Key Methods to Implement:
func NewRegistry(manifestPath string) *Registry
func (r *Registry) LoadManifest() error
func (r *Registry) SaveManifest() error
func (r *Registry) RegisterGadget(metadata *GadgetMetadata) error
func (r *Registry) UnregisterGadget(name string) error
func (r *Registry) FindGadget(name string) (*GadgetMetadata, error)
func (r *Registry) SearchGadgets(query string) []*GadgetMetadata
func (r *Registry) ListByCategory(category GadgetCategory) []*GadgetMetadata
```

## 🔒 Security Specifications

### Permission System (`internal/security/permissions.go`)

```go
// Permission defines what a gadget is allowed to do
type Permission string

const (
    PermissionFileRead     Permission = "file:read"
    PermissionFileWrite    Permission = "file:write"
    PermissionFileExecute  Permission = "file:execute"
    PermissionNetworkRead  Permission = "network:read"
    PermissionNetworkWrite Permission = "network:write"
    PermissionSystemInfo   Permission = "system:info"
    PermissionSystemExec   Permission = "system:exec"
    PermissionAIAccess     Permission = "ai:access"
    PermissionDBRead       Permission = "database:read"
    PermissionDBWrite      Permission = "database:write"
)

// PermissionManager validates gadget permissions
type PermissionManager struct {
    userPermissions map[string][]Permission
    gadgetPermissions map[string][]Permission
}

// Key Methods:
func (pm *PermissionManager) ValidateExecution(gadgetName, userID string, requiredPerms []Permission) error
func (pm *PermissionManager) GrantPermission(gadgetName string, perm Permission) error
func (pm *PermissionManager) RevokePermission(gadgetName string, perm Permission) error
```

## 🧪 Testing Requirements

### Test Coverage Goals
- **Unit Tests**: >90% coverage for core components
- **Integration Tests**: Full gadget lifecycle testing
- **Performance Tests**: Load testing with multiple gadgets
- **Security Tests**: Permission boundary validation

### Specific Test Cases

1. **Interface Tests** (`gadget/interface_test.go`):
   - Gadget lifecycle (Initialize -> Start -> Execute -> Stop)
   - Error handling for invalid configurations
   - Health check responses

2. **Manager Tests** (`gadget/manager_test.go`):
   - Concurrent gadget loading/unloading
   - Dependency resolution
   - Resource cleanup on shutdown
   - Error recovery scenarios

3. **Parser Tests** (`command/parser_test.go`):
   - Various "Go Go Gadget" command formats
   - Edge cases and malformed commands
   - Context extraction and validation
   - Command suggestions

4. **Registry Tests** (`gadget/registry/registry_test.go`):
   - Manifest loading/saving
   - Gadget discovery and search
   - Category filtering
   - Metadata validation

## 📝 Example Implementations

### Simple Hello Gadget (`examples/hello_gadget.go`)

```go
type HelloGadget struct {
    config Config
    logger Logger
}

func (h *HelloGadget) Name() string { return "Hello World" }
func (h *HelloGadget) Category() GadgetCategory { return CategoryCustom }

func (h *HelloGadget) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    name := "World"
    if len(req.Args) > 0 {
        name = req.Args[0]
    }
    
    return &ExecuteResponse{
        Success:  true,
        Output:   fmt.Sprintf("Hello, %s!", name),
        ExitCode: 0,
    }, nil
}

// Usage: "Go Go Gadget Hello Alice" -> "Hello, Alice!"
```

### Weather Integration (`examples/weather_integration.go`)

```go
// Integrate the existing weather gadget from gadgets/examples/weather/main.go
// Should demonstrate:
// - Loading external gadgets
// - API integration
// - Error handling
// - Configuration management
```

## 🔧 Configuration

### Gadget Manifest (`gadgets/manifest.yaml`)

```yaml
version: "1.0"
gadgets:
  hello-world:
    name: "Hello World"
    description: "Simple greeting gadget"
    category: "custom"
    version: "1.0.0"
    author: "Inspector Gadget Team"
    binary: "./examples/hello_gadget.so"
    permissions:
      - "system:info"
    
  weather:
    name: "Weather Report"
    description: "Get weather information for any location"
    category: "productivity" 
    version: "1.0.0"
    author: "Inspector Gadget Team"
    binary: "./examples/weather/weather.so"
    permissions:
      - "network:read"
    config:
      api_key: "${WEATHER_API_KEY}"
      
  network-scanner:
    name: "Network Scanner"
    description: "AI-enhanced network reconnaissance"
    category: "security"
    version: "1.2.0"
    binary: "./security/network-scanner.so"
    permissions:
      - "network:read"
      - "network:write"
      - "system:exec"
```

## 📚 Documentation Requirements

1. **README.md** - Overview and quick start guide
2. **API.md** - Complete API documentation
3. **DEVELOPMENT.md** - Guide for creating new gadgets
4. **EXAMPLES.md** - Usage examples and tutorials

## ✅ Acceptance Criteria

### Phase 1: Core Framework
- [ ] All interfaces defined with comprehensive documentation
- [ ] GadgetManager can load/unload gadgets dynamically
- [ ] Registry system discovers and manages gadget metadata
- [ ] Command parser handles basic "Go Go Gadget" patterns
- [ ] Permission system validates gadget access
- [ ] All core components have >90% test coverage

### Phase 2: Integration
- [ ] Hello World gadget loads and executes successfully
- [ ] Weather gadget integration works end-to-end
- [ ] CLI tool can list, load, and execute gadgets
- [ ] Configuration system loads from YAML manifest
- [ ] Error handling and recovery mechanisms work properly

### Phase 3: Polish
- [ ] Comprehensive documentation complete
- [ ] Performance benchmarks pass (100+ concurrent gadgets)
- [ ] Security audit shows no permission leaks
- [ ] Build system integration (`make gadget-cli` works)
- [ ] Ready for future O-LLaMA AI integration

## 🚀 Integration Points

### Future O-LLaMA Integration
The framework should be designed to easily integrate with:
- JWT authentication system (for user context)
- SafeFS operations (for file access)
- AI-powered command understanding
- MCP server communication

### Build System Integration
- Must work with existing Makefile
- Should produce `bin/go-go-gadget` executable
- Support for `make test` and `make clean`

## 📋 Deliverable Checklist

- [ ] All source files created with proper package structure
- [ ] Comprehensive test suite with >90% coverage
- [ ] Example gadgets (Hello World + Weather integration)
- [ ] CLI tool for gadget management
- [ ] YAML manifest system
- [ ] Documentation (README, API docs, examples)
- [ ] Integration with existing build system
- [ ] Permission and security validation
- [ ] Error handling and logging throughout
- [ ] Performance considerations for concurrent access

---

This specification provides a complete, independent task that won't interfere with our O-LLaMA development. The implementer can work entirely in the `gadget-framework/` directory while we continue with Casbin RBAC and MCP integration.

**Estimated Timeline**: 1-2 weeks for a skilled Go developer
**Complexity**: Intermediate to Advanced
**Dependencies**: None (completely independent)