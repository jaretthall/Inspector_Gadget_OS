// Package mcp provides MCP server discovery and management for O-LLaMA
package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MCPManager manages multiple MCP server connections
type MCPManager struct {
	clients       map[string]*MCPClient
	configs       map[string]*MCPServerConfig
	factory       *TransportFactory
	logger        *log.Logger
	mutex         sync.RWMutex
	
	// Health monitoring
	healthCheck   time.Duration
	healthTicker  *time.Ticker
	healthStop    chan struct{}
	
	// Event handlers
	onServerConnect    func(string, *ServerInfo)
	onServerDisconnect func(string, error)
	onResourceChange   func(string, []Resource)
	onToolChange       func(string, []Tool)
}

// MCPServerConfig holds configuration for an MCP server
type MCPServerConfig struct {
	Name         string                 `json:"name" yaml:"name"`
	Description  string                 `json:"description" yaml:"description"`
	Transport    map[string]interface{} `json:"transport" yaml:"transport"`
	AutoStart    bool                   `json:"auto_start" yaml:"auto_start"`
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	Timeout      time.Duration          `json:"timeout" yaml:"timeout"`
	RetryCount   int                    `json:"retry_count" yaml:"retry_count"`
	RetryDelay   time.Duration          `json:"retry_delay" yaml:"retry_delay"`
	Environment  map[string]string      `json:"environment" yaml:"environment"`
}

// MCPManagerConfig holds configuration for the MCP manager
type MCPManagerConfig struct {
	Servers      map[string]*MCPServerConfig `json:"servers" yaml:"servers"`
	ClientName   string                      `json:"client_name" yaml:"client_name"`
	ClientVersion string                     `json:"client_version" yaml:"client_version"`
	HealthCheck  time.Duration              `json:"health_check" yaml:"health_check"`
	Logger       *log.Logger
}

// ServerStatus represents the current status of a server
type ServerStatus struct {
	Name         string              `json:"name"`
	Connected    bool                `json:"connected"`
	Initialized  bool                `json:"initialized"`
	LastConnected time.Time          `json:"last_connected,omitempty"`
	LastError    string              `json:"last_error,omitempty"`
	ServerInfo   *ServerInfo         `json:"server_info,omitempty"`
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`
	Resources    []Resource          `json:"resources,omitempty"`
	Tools        []Tool              `json:"tools,omitempty"`
	Prompts      []Prompt            `json:"prompts,omitempty"`
}

// NewMCPManager creates a new MCP manager
func NewMCPManager(config MCPManagerConfig) *MCPManager {
	if config.Logger == nil {
		config.Logger = log.Default()
	}
	
	if config.HealthCheck == 0 {
		config.HealthCheck = 30 * time.Second
	}
	
	if config.ClientName == "" {
		config.ClientName = "o-llama"
	}
	
	if config.ClientVersion == "" {
		config.ClientVersion = "1.0.0"
	}
	
	return &MCPManager{
		clients:     make(map[string]*MCPClient),
		configs:     config.Servers,
		factory:     &TransportFactory{},
		logger:      config.Logger,
		healthCheck: config.HealthCheck,
		healthStop:  make(chan struct{}),
	}
}

// Start starts the MCP manager and auto-connects to enabled servers
func (m *MCPManager) Start(ctx context.Context) error {
	m.logger.Printf("Starting MCP manager with %d configured servers", len(m.configs))
	
	// Start health monitoring
	m.startHealthMonitoring()
	
	// Auto-connect to enabled servers
	for name, config := range m.configs {
		if config.Enabled && config.AutoStart {
			if err := m.ConnectServer(ctx, name); err != nil {
				m.logger.Printf("Failed to auto-connect to server %s: %v", name, err)
			}
		}
	}
	
	return nil
}

// ConnectServer establishes connection to a specific MCP server
func (m *MCPManager) ConnectServer(ctx context.Context, serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	config, exists := m.configs[serverName]
	if !exists {
		return fmt.Errorf("server %s not configured", serverName)
	}
	
	if !config.Enabled {
		return fmt.Errorf("server %s is disabled", serverName)
	}
	
	// Check if already connected
	if client, exists := m.clients[serverName]; exists && client.IsConnected() {
		return fmt.Errorf("server %s already connected", serverName)
	}
	
	// Create transport
	transport, err := m.factory.CreateTransport(config.Transport)
	if err != nil {
		return fmt.Errorf("failed to create transport for %s: %w", serverName, err)
	}
	
	// Connect transport
	if err := transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect transport for %s: %w", serverName, err)
	}
	
	// Create and configure client
	clientConfig := MCPClientConfig{
		Name:         fmt.Sprintf("o-llama-%s", serverName),
		Version:      "1.0.0",
		Transport:    transport,
		Capabilities: ClientCapabilities{},
		Logger:       m.logger,
		Timeout:      config.Timeout,
	}
	
	client := NewMCPClient(clientConfig)
	
	// Set up event handlers
	client.SetResourceChangeHandler(func(resources []Resource) {
		if m.onResourceChange != nil {
			m.onResourceChange(serverName, resources)
		}
	})
	
	client.SetToolChangeHandler(func(tools []Tool) {
		if m.onToolChange != nil {
			m.onToolChange(serverName, tools)
		}
	})
	
	// Connect client
	if err := client.Connect(ctx); err != nil {
		transport.Close()
		return fmt.Errorf("failed to connect MCP client for %s: %w", serverName, err)
	}
	
	// Store client
	m.clients[serverName] = client
	
	m.logger.Printf("Successfully connected to MCP server: %s", serverName)
	
	// Trigger connection callback
	if m.onServerConnect != nil {
		m.onServerConnect(serverName, client.GetServerInfo())
	}
	
	return nil
}

// DisconnectServer disconnects from a specific MCP server
func (m *MCPManager) DisconnectServer(serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	client, exists := m.clients[serverName]
	if !exists {
		return fmt.Errorf("server %s not connected", serverName)
	}
	
	err := client.Close()
	delete(m.clients, serverName)
	
	m.logger.Printf("Disconnected from MCP server: %s", serverName)
	
	// Trigger disconnection callback
	if m.onServerDisconnect != nil {
		m.onServerDisconnect(serverName, err)
	}
	
	return err
}

// GetClient returns the MCP client for a server
func (m *MCPManager) GetClient(serverName string) (*MCPClient, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	client, exists := m.clients[serverName]
	if !exists {
		return nil, fmt.Errorf("server %s not connected", serverName)
	}
	
	if !client.IsReady() {
		return nil, fmt.Errorf("server %s not ready", serverName)
	}
	
	return client, nil
}

// ListResources returns all resources from all connected servers
func (m *MCPManager) ListResources(ctx context.Context) (map[string][]Resource, error) {
	m.mutex.RLock()
	clients := make(map[string]*MCPClient)
	for name, client := range m.clients {
		if client.IsReady() {
			clients[name] = client
		}
	}
	m.mutex.RUnlock()
	
	resources := make(map[string][]Resource)
	
	for serverName, client := range clients {
		response, err := client.ListResources(ctx)
		if err != nil {
			m.logger.Printf("Failed to list resources from %s: %v", serverName, err)
			continue
		}
		
		resources[serverName] = response.Resources
	}
	
	return resources, nil
}

// ListTools returns all tools from all connected servers
func (m *MCPManager) ListTools(ctx context.Context) (map[string][]Tool, error) {
	m.mutex.RLock()
	clients := make(map[string]*MCPClient)
	for name, client := range m.clients {
		if client.IsReady() {
			clients[name] = client
		}
	}
	m.mutex.RUnlock()
	
	tools := make(map[string][]Tool)
	
	for serverName, client := range clients {
		response, err := client.ListTools(ctx)
		if err != nil {
			m.logger.Printf("Failed to list tools from %s: %v", serverName, err)
			continue
		}
		
		tools[serverName] = response.Tools
	}
	
	return tools, nil
}

// CallTool calls a tool on a specific server
func (m *MCPManager) CallTool(ctx context.Context, serverName, toolName string, arguments interface{}) (*CallToolResponse, error) {
	client, err := m.GetClient(serverName)
	if err != nil {
		return nil, err
	}
	
	return client.CallTool(ctx, toolName, arguments)
}

// ReadResource reads a resource from a specific server
func (m *MCPManager) ReadResource(ctx context.Context, serverName, uri string) (*ReadResourceResponse, error) {
	client, err := m.GetClient(serverName)
	if err != nil {
		return nil, err
	}
	
	return client.ReadResource(ctx, uri)
}

// GetServerStatus returns status information for all servers
func (m *MCPManager) GetServerStatus() map[string]*ServerStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	status := make(map[string]*ServerStatus)
	
	for serverName := range m.configs {
		serverStatus := &ServerStatus{
			Name:      serverName,
			Connected: false,
		}
		
		if client, exists := m.clients[serverName]; exists {
			serverStatus.Connected = client.IsConnected()
			serverStatus.Initialized = client.IsReady()
			serverStatus.ServerInfo = client.GetServerInfo()
			serverStatus.Capabilities = client.GetServerCapabilities()
			
			// Get resources, tools, and prompts
			statusCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			if resources, err := client.ListResources(statusCtx); err == nil {
				serverStatus.Resources = resources.Resources
			}
			
			if tools, err := client.ListTools(statusCtx); err == nil {
				serverStatus.Tools = tools.Tools
			}
			
			if prompts, err := client.ListPrompts(statusCtx); err == nil {
				serverStatus.Prompts = prompts.Prompts
			}
		}
		
		status[serverName] = serverStatus
	}
	
	return status
}

// SetServerConnectHandler sets callback for server connections
func (m *MCPManager) SetServerConnectHandler(handler func(string, *ServerInfo)) {
	m.onServerConnect = handler
}

// SetServerDisconnectHandler sets callback for server disconnections
func (m *MCPManager) SetServerDisconnectHandler(handler func(string, error)) {
	m.onServerDisconnect = handler
}

// SetResourceChangeHandler sets callback for resource changes
func (m *MCPManager) SetResourceChangeHandler(handler func(string, []Resource)) {
	m.onResourceChange = handler
}

// SetToolChangeHandler sets callback for tool changes
func (m *MCPManager) SetToolChangeHandler(handler func(string, []Tool)) {
	m.onToolChange = handler
}

// startHealthMonitoring starts the health monitoring routine
func (m *MCPManager) startHealthMonitoring() {
	m.healthTicker = time.NewTicker(m.healthCheck)
	
	go func() {
		for {
			select {
			case <-m.healthTicker.C:
				m.performHealthCheck()
			case <-m.healthStop:
				return
			}
		}
	}()
}

// performHealthCheck checks the health of all connections
func (m *MCPManager) performHealthCheck() {
	m.mutex.RLock()
	clients := make(map[string]*MCPClient)
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mutex.RUnlock()
	
	for serverName, client := range clients {
		if !client.IsConnected() {
			m.logger.Printf("Health check: server %s disconnected", serverName)
			
			// Attempt to reconnect if auto-start is enabled
			if serverConfig, exists := m.configs[serverName]; exists && serverConfig.AutoStart {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := m.ConnectServer(ctx, serverName); err != nil {
					m.logger.Printf("Failed to reconnect to %s: %v", serverName, err)
				}
				cancel()
			}
		}
	}
}

// Stop stops the MCP manager and disconnects all servers
func (m *MCPManager) Stop() error {
	m.logger.Printf("Stopping MCP manager")
	
	// Stop health monitoring
	if m.healthTicker != nil {
		m.healthTicker.Stop()
		close(m.healthStop)
	}
	
	// Disconnect all servers
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for serverName := range m.clients {
		if err := m.DisconnectServer(serverName); err != nil {
			m.logger.Printf("Error disconnecting from %s: %v", serverName, err)
		}
	}
	
	return nil
}

// AddServer adds a new server configuration
func (m *MCPManager) AddServer(name string, config *MCPServerConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.configs[name] = config
	m.logger.Printf("Added MCP server configuration: %s", name)
}

// RemoveServer removes a server configuration and disconnects if connected
func (m *MCPManager) RemoveServer(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Disconnect if connected
	if client, exists := m.clients[name]; exists {
		client.Close()
		delete(m.clients, name)
	}
	
	// Remove configuration
	delete(m.configs, name)
	
	m.logger.Printf("Removed MCP server: %s", name)
	return nil
}

// GetServerConfigs returns all server configurations
func (m *MCPManager) GetServerConfigs() map[string]*MCPServerConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	configs := make(map[string]*MCPServerConfig)
	for name, config := range m.configs {
		configs[name] = config
	}
	
	return configs
}