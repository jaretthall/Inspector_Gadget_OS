// Package mcp provides MCP client implementation for O-LLaMA
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// MCPClient represents an MCP protocol client
type MCPClient struct {
	name         string
	version      string
	transport    Transport
	capabilities ClientCapabilities
	serverInfo   *ServerInfo
	serverCaps   *ServerCapabilities
	idGen        *MessageIDGenerator
	
	// Connection state
	connected    bool
	initialized  bool
	mutex        sync.RWMutex
	
	// Request tracking
	pendingRequests map[interface{}]chan *Message
	requestMutex    sync.RWMutex
	
	// Event handlers
	onResourceChanged func([]Resource)
	onToolChanged     func([]Tool)
	onPromptChanged   func([]Prompt)
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	
	logger *log.Logger
}

// MCPClientConfig holds configuration for MCP client
type MCPClientConfig struct {
	Name         string
	Version      string
	Transport    Transport
	Capabilities ClientCapabilities
	Logger       *log.Logger
	Timeout      time.Duration
}

// NewMCPClient creates a new MCP client
func NewMCPClient(config MCPClientConfig) *MCPClient {
	ctx, cancel := context.WithCancel(context.Background())
	
	if config.Logger == nil {
		config.Logger = log.Default()
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &MCPClient{
		name:            config.Name,
		version:         config.Version,
		transport:       config.Transport,
		capabilities:    config.Capabilities,
		idGen:           NewMessageIDGenerator(),
		pendingRequests: make(map[interface{}]chan *Message),
		ctx:             ctx,
		cancel:          cancel,
		logger:          config.Logger,
	}
}

// Connect establishes connection and initializes the MCP session
func (c *MCPClient) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.connected {
		return fmt.Errorf("client already connected")
	}
	
	// Check transport connection
	if !c.transport.IsConnected() {
		return fmt.Errorf("transport not connected")
	}
	
	// Start message handling
	go c.handleMessages()
	
	// Send initialize request
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	
	c.connected = true
	c.logger.Printf("MCP client connected and initialized")
	
	return nil
}

// initialize performs the MCP initialization handshake
func (c *MCPClient) initialize(ctx context.Context) error {
	initRequest := InitializeRequest{
		ProtocolVersion: MCPVersion,
		Capabilities:    c.capabilities,
		ClientInfo: ClientInfo{
			Name:    c.name,
			Version: c.version,
		},
	}
	
	response, err := c.sendRequest(ctx, "initialize", initRequest)
	if err != nil {
		return err
	}
	
	if response.Error != nil {
		return fmt.Errorf("initialization error: %s", response.Error.Message)
	}
	
	// Parse initialize response
	var initResponse InitializeResponse
	if err := parseResult(response.Result, &initResponse); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}
	
	// Validate protocol version
	if initResponse.ProtocolVersion != MCPVersion {
		return fmt.Errorf("protocol version mismatch: client=%s, server=%s", 
			MCPVersion, initResponse.ProtocolVersion)
	}
	
	// Store server information
	c.serverInfo = &initResponse.ServerInfo
	c.serverCaps = &initResponse.Capabilities
	c.initialized = true
	
	// Send initialized notification
	notification := CreateNotification("notifications/initialized", nil)
	return c.transport.Send(notification)
}

// sendRequest sends a request and waits for response
func (c *MCPClient) sendRequest(ctx context.Context, method string, params interface{}) (*Message, error) {
	if !c.transport.IsConnected() {
		return nil, fmt.Errorf("transport not connected")
	}
	
	// Generate request ID
	id := c.idGen.Next()
	
	// Create response channel
	respChan := make(chan *Message, 1)
	c.requestMutex.Lock()
	c.pendingRequests[id] = respChan
	c.requestMutex.Unlock()
	
	// Cleanup on exit
	defer func() {
		c.requestMutex.Lock()
		delete(c.pendingRequests, id)
		close(respChan)
		c.requestMutex.Unlock()
	}()
	
	// Create and send request
	request := CreateRequest(id, method, params)
	if err := c.transport.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Wait for response with timeout
	select {
	case response := <-respChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// handleMessages processes incoming messages
func (c *MCPClient) handleMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			message, err := c.transport.Receive()
			if err != nil {
				c.logger.Printf("Error receiving message: %v", err)
				continue
			}
			
			if err := ValidateMessage(message); err != nil {
				c.logger.Printf("Invalid message: %v", err)
				continue
			}
			
			c.processMessage(message)
		}
	}
}

// processMessage processes a received message
func (c *MCPClient) processMessage(message *Message) {
	if message.IsResponse() {
		c.handleResponse(message)
	} else if message.IsNotification() {
		c.handleNotification(message)
	} else if message.IsRequest() {
		c.handleRequest(message)
	}
}

// handleResponse handles response messages
func (c *MCPClient) handleResponse(message *Message) {
	c.requestMutex.RLock()
	respChan, exists := c.pendingRequests[message.ID]
	c.requestMutex.RUnlock()
	
	if exists {
		select {
		case respChan <- message:
		default:
			c.logger.Printf("Response channel full for request %v", message.ID)
		}
	} else {
		c.logger.Printf("Received response for unknown request ID: %v", message.ID)
	}
}

// handleNotification handles notification messages
func (c *MCPClient) handleNotification(message *Message) {
	switch message.Method {
	case "notifications/resources/list_changed":
		if c.onResourceChanged != nil {
			// Fetch updated resources
			if resources, err := c.ListResources(c.ctx); err == nil {
				c.onResourceChanged(resources.Resources)
			}
		}
		
	case "notifications/tools/list_changed":
		if c.onToolChanged != nil {
			// Fetch updated tools
			if tools, err := c.ListTools(c.ctx); err == nil {
				c.onToolChanged(tools.Tools)
			}
		}
		
	case "notifications/prompts/list_changed":
		if c.onPromptChanged != nil {
			// Fetch updated prompts
			if prompts, err := c.ListPrompts(c.ctx); err == nil {
				c.onPromptChanged(prompts.Prompts)
			}
		}
		
	default:
		c.logger.Printf("Unknown notification: %s", message.Method)
	}
}

// handleRequest handles request messages (from server to client)
func (c *MCPClient) handleRequest(message *Message) {
	// For now, we don't handle server-to-client requests
	// This would be used for sampling/completion requests
	errorResp := CreateErrorResponse(message.ID, MethodNotFound, 
		"Method not implemented", nil)
	c.transport.Send(errorResp)
}

// ListResources requests available resources from the server
func (c *MCPClient) ListResources(ctx context.Context) (*ListResourcesResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	response, err := c.sendRequest(ctx, "resources/list", nil)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result ListResourcesResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// ReadResource requests content of a specific resource
func (c *MCPClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	request := ReadResourceRequest{URI: uri}
	response, err := c.sendRequest(ctx, "resources/read", request)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result ReadResourceResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// ListTools requests available tools from the server
func (c *MCPClient) ListTools(ctx context.Context) (*ListToolsResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	response, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result ListToolsResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// CallTool executes a tool on the server
func (c *MCPClient) CallTool(ctx context.Context, name string, arguments interface{}) (*CallToolResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	request := CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}
	
	response, err := c.sendRequest(ctx, "tools/call", request)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result CallToolResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// ListPrompts requests available prompts from the server
func (c *MCPClient) ListPrompts(ctx context.Context) (*ListPromptsResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	response, err := c.sendRequest(ctx, "prompts/list", nil)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result ListPromptsResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// GetPrompt requests a specific prompt from the server
func (c *MCPClient) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*GetPromptResponse, error) {
	if !c.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}
	
	request := GetPromptRequest{
		Name:      name,
		Arguments: arguments,
	}
	
	response, err := c.sendRequest(ctx, "prompts/get", request)
	if err != nil {
		return nil, err
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)
	}
	
	var result GetPromptResponse
	if err := parseResult(response.Result, &result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// SetResourceChangeHandler sets callback for resource list changes
func (c *MCPClient) SetResourceChangeHandler(handler func([]Resource)) {
	c.onResourceChanged = handler
}

// SetToolChangeHandler sets callback for tool list changes
func (c *MCPClient) SetToolChangeHandler(handler func([]Tool)) {
	c.onToolChanged = handler
}

// SetPromptChangeHandler sets callback for prompt list changes
func (c *MCPClient) SetPromptChangeHandler(handler func([]Prompt)) {
	c.onPromptChanged = handler
}

// IsConnected returns whether the transport is connected
func (c *MCPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.transport.IsConnected()
}

// IsReady returns whether the client is ready for operations
func (c *MCPClient) IsReady() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.initialized && c.transport.IsConnected()
}

// GetServerInfo returns information about the connected server
func (c *MCPClient) GetServerInfo() *ServerInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.serverInfo
}

// GetServerCapabilities returns the server's capabilities
func (c *MCPClient) GetServerCapabilities() *ServerCapabilities {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.serverCaps
}

// Close closes the connection and cleans up resources
func (c *MCPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return nil
	}
	
	// Cancel context to stop message handling
	c.cancel()
	
	// Close transport
	err := c.transport.Close()
	
	// Clear pending requests
	c.requestMutex.Lock()
	for _, ch := range c.pendingRequests {
		close(ch)
	}
	c.pendingRequests = make(map[interface{}]chan *Message)
	c.requestMutex.Unlock()
	
	c.connected = false
	c.initialized = false
	
	c.logger.Printf("MCP client disconnected")
	return err
}

// parseResult parses JSON result into a struct
func parseResult(result interface{}, target interface{}) error {
	if result == nil {
		return fmt.Errorf("nil result")
	}
	
	// Convert through JSON to handle type conversion
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(jsonBytes, target)
}