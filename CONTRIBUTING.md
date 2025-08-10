# Contributing to Inspector Gadget OS

Welcome to the Inspector Gadget OS community! We're building the ultimate AI Swiss Army knife, and we need your help to make it amazing. Whether you're contributing code, documentation, gadgets, or ideas, every contribution matters.

## =€ Getting Started

### Development Environment Setup
Follow the [DEVELOPMENT.md](./DEVELOPMENT.md) guide to set up your development environment.

### Ways to Contribute
- **=' Core Development**: Work on O-LLaMA, gadget framework, or OS components
- **<¯ Gadget Creation**: Build new gadgets for the community
- **=ñ Web UI**: Improve the management interface
- **=Ú Documentation**: Help others understand and use the system
- **= Bug Reports**: Help us identify and fix issues
- **=¡ Feature Requests**: Suggest new capabilities and improvements

## <¯ Gadget Development

### Creating Your First Gadget
Gadgets are the heart of Inspector Gadget OS. Here's how to create one:

1. **Generate scaffold:**
```bash
cd gadget-framework
./go-go-gadget create weather-assistant
```

2. **Implement the Gadget interface:**
```go
// gadgets/weather-assistant/main.go
package main

import (
    "context"
    "github.com/inspector-gadget/core/gadget"
)

type WeatherGadget struct {
    apiKey string
    ai     gadget.AIEngine
}

func (w *WeatherGadget) Name() string { return "Weather Assistant" }
func (w *WeatherGadget) Category() gadget.GadgetCategory { return gadget.CategoryProductivity }
func (w *WeatherGadget) Description() string { 
    return "AI-powered weather analysis with actionable insights" 
}

func (w *WeatherGadget) Execute(ctx context.Context, args []string) (*gadget.Result, error) {
    // Your implementation here
    return &gadget.Result{
        Type:    "weather_report",
        Content: "Today will be sunny with a high of 75°F. Perfect for outdoor activities!",
    }, nil
}

// Usage: "Go Go Gadget Weather" or "Go Go Gadget Weather New York"
```

3. **Test your gadget:**
```bash
go test ./gadgets/weather-assistant/...
./go-go-gadget test weather-assistant
```

### Gadget Guidelines
- **Security First**: Follow security best practices, validate all inputs
- **AI Integration**: Enhance functionality with AI where appropriate
- **User Experience**: Provide helpful error messages and guidance
- **Documentation**: Include clear usage examples and API documentation
- **Privacy**: Respect user data and provide transparent data handling

### Gadget Categories
- **> AI Gadgets**: LLM integration, conversation, analysis
- **=á Security Gadgets**: Network scanning, vulnerability assessment, forensics
- **¡ Productivity Gadgets**: Task management, development tools, automation
- **=' System Gadgets**: Hardware monitoring, performance, maintenance
- **<¯ Custom Gadgets**: Your unique creations and integrations

## =' Core Development

### Component Architecture
- **O-LLaMA**: Enhanced Ollama with file access, auth, and MCP integration
- **Gadget Framework**: Plugin system with lifecycle management
- **OS Core**: Alpine Linux base with immutable updates
- **Web UI**: React-based management interface
- **Security Layer**: JWT auth, RBAC, and sandboxing

### Development Process
1. **Create feature branch**: `git checkout -b feature/amazing-feature`
2. **Follow coding standards**: See style guide below
3. **Add comprehensive tests**: Unit, integration, and security tests
4. **Update documentation**: Keep docs current with changes
5. **Submit pull request**: Use PR template and request review

### Code Style Guidelines

**Go Code:**
```go
// Use meaningful names
func (g *GadgetManager) InstallGadget(name string) error

// Document public APIs
// InstallGadget downloads and installs a gadget from the registry
func (g *GadgetManager) InstallGadget(name string) error {
    // Validate input
    if name == "" {
        return fmt.Errorf("gadget name cannot be empty")
    }
    
    // Implementation...
}

// Use early returns
func validatePath(path string) error {
    if path == "" {
        return ErrEmptyPath
    }
    if strings.Contains(path, "..") {
        return ErrPathTraversal
    }
    return nil
}
```

**TypeScript/React:**
```typescript
// Use functional components with hooks
export const GadgetList: React.FC = () => {
  const [gadgets, setGadgets] = useState<Gadget[]>([]);
  
  useEffect(() => {
    loadGadgets();
  }, []);
  
  return (
    <div className="gadget-grid">
      {gadgets.map(gadget => (
        <GadgetCard key={gadget.id} gadget={gadget} />
      ))}
    </div>
  );
};

// Use proper TypeScript types
interface GadgetConfig {
  name: string;
  enabled: boolean;
  settings: Record<string, unknown>;
}
```

## >ê Testing Requirements

### Unit Tests
- **Coverage**: Minimum 80% code coverage
- **Scope**: Test individual functions and methods
- **Mocking**: Mock external dependencies

```go
func TestGadgetManager_InstallGadget(t *testing.T) {
    tests := []struct {
        name      string
        gadgetName string
        want      error
    }{
        {"Valid gadget", "network-scanner", nil},
        {"Empty name", "", ErrEmptyGadgetName},
        {"Invalid name", "../etc/passwd", ErrInvalidGadgetName},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gm := NewGadgetManager()
            err := gm.InstallGadget(tt.gadgetName)
            assert.Equal(t, tt.want, err)
        })
    }
}
```

### Integration Tests
- **API Testing**: Test complete request/response cycles
- **Component Integration**: Test component interactions
- **End-to-End**: Test user workflows

### Security Tests
- **Path Traversal**: Validate file path security
- **Authentication**: Test auth bypass attempts
- **Input Validation**: Test with malicious inputs
- **Resource Limits**: Test resource exhaustion

## = Security Requirements

### Defensive Security Only
We only accept contributions for **defensive security** purposes:
-  Vulnerability scanners and assessment tools
-  Network monitoring and intrusion detection
-  Security analysis and reporting
-  Educational security demonstrations
- L Exploitation tools or malicious code
- L Attack frameworks or offensive capabilities

### Security Best Practices
- **Input Validation**: Validate and sanitize all user inputs
- **Path Security**: Use `safefs` package for file operations
- **Authentication**: Require auth for sensitive operations
- **Audit Logging**: Log all security-relevant actions
- **Least Privilege**: Run with minimal required permissions

## =Ú Documentation Standards

### Code Documentation
```go
// Package gadget provides the core interfaces for Inspector Gadget OS plugins.
//
// Gadgets are modular components that extend the system's capabilities.
// Each gadget implements the Gadget interface and can be dynamically loaded.
package gadget

// Gadget represents a modular component that provides specific functionality.
// Gadgets can be AI tools, security utilities, productivity apps, or system monitors.
type Gadget interface {
    // Name returns the human-readable name of the gadget.
    Name() string
    
    // Execute runs the gadget with the provided arguments and context.
    // It returns a Result containing the output and any actions to perform.
    Execute(ctx context.Context, args []string) (*Result, error)
}
```

### User Documentation
- **Clear Examples**: Provide working code examples
- **Use Cases**: Show real-world applications
- **Troubleshooting**: Include common issues and solutions
- **Screenshots**: Visual documentation where helpful

## <÷ Issue and PR Guidelines

### Bug Reports
Use the bug report template and include:
- **Environment**: OS, version, hardware details
- **Steps to Reproduce**: Clear, numbered steps
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Logs**: Relevant log excerpts with stack traces

### Feature Requests
- **Use Case**: Why is this feature needed?
- **Proposed Solution**: How would you implement it?
- **Alternatives**: What other approaches were considered?
- **Breaking Changes**: Will this impact existing users?

### Pull Request Template
```markdown
## Description
Brief description of changes and why they're needed.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality) 
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed
- [ ] Security testing completed (if applicable)

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes without discussion
```

## < Recognition

### Contributor Types
- **=' Core Contributors**: Regular contributors to core codebase
- **<¯ Gadget Creators**: Developers of popular community gadgets
- **=Ú Documentation Heroes**: Contributors who improve documentation
- **= Bug Hunters**: Contributors who find and fix issues
- **<¨ UI/UX Designers**: Contributors who improve user experience

### Hall of Fame
Outstanding contributors are recognized in our:
- README.md contributor section
- Monthly community newsletters
- Conference speaking opportunities
- Exclusive contributor Discord channels

## > Community Guidelines

### Code of Conduct
- **Be Respectful**: Treat everyone with kindness and professionalism
- **Be Inclusive**: Welcome contributors from all backgrounds
- **Be Constructive**: Provide helpful feedback and suggestions
- **Be Patient**: Remember that everyone is learning

### Communication Channels
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General development discussions
- **Discord**: Real-time chat and community support
- **Monthly Calls**: Virtual meetups for major contributors

### Getting Help
1. **Search First**: Check existing issues and documentation
2. **Ask Questions**: Don't hesitate to ask in Discord or GitHub Discussions
3. **Be Specific**: Provide context, code snippets, and error messages
4. **Follow Up**: Update on your progress and solutions found

## <¯ Project Roadmap Participation

### Current Phase: Phase 2 - O-LLaMA Development (Weeks 3-4)
Priority contributions needed:
- Enhanced Ollama fork with file system access
- JWT authentication and RBAC implementation
- MCP integration for tool execution
- Comprehensive security testing

### Upcoming Phases
- **Phase 3**: Base OS development (Alpine Linux, atomic updates)
- **Phase 4**: Kali tools integration (security gadgets)
- **Phase 5**: Ultron assistant integration (productivity)
- **Phase 6**: Advanced gadget framework

### How to Get Involved
1. **Join Planning**: Participate in roadmap discussions
2. **Claim Tasks**: Pick up issues tagged with current phase
3. **Propose Features**: Suggest improvements for upcoming phases
4. **Beta Testing**: Help test new releases and features

---

**Ready to join the Inspector Gadget OS community?** >=€

Start by setting up your development environment and creating your first gadget. Every contribution, no matter how small, helps build the ultimate AI Swiss Army knife!

**Questions?** Join our [Discord](https://discord.gg/inspector-gadget) or start a [GitHub Discussion](https://github.com/your-org/inspector-gadget-os/discussions).