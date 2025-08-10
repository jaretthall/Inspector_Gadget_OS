// Package mcp implements the Model Context Protocol for O-LLaMA integration
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// MCPVersion represents the MCP protocol version
const MCPVersion = "2024-11-05"

// Transport defines the interface for MCP communication mechanisms
type Transport interface {
	Connect(ctx context.Context) error
	Send(message *Message) error
	Receive() (*Message, error)
	Close() error
	IsConnected() bool
}

// Message represents a JSON-RPC 2.0 message
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Common RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// ClientCapabilities represents what the client supports
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     *SamplingCapabilities  `json:"sampling,omitempty"`
}

// ServerCapabilities represents what the server supports
type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      *LoggingCapabilities   `json:"logging,omitempty"`
	Prompts      *PromptCapabilities    `json:"prompts,omitempty"`
	Resources    *ResourceCapabilities  `json:"resources,omitempty"`
	Tools        *ToolCapabilities      `json:"tools,omitempty"`
}

// SamplingCapabilities defines sampling feature support
type SamplingCapabilities struct {
	// No specific capabilities defined in current spec
}

// LoggingCapabilities defines logging feature support
type LoggingCapabilities struct {
	// No specific capabilities defined in current spec
}

// PromptCapabilities defines prompt feature support
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities defines resource feature support
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolCapabilities defines tool feature support
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeRequest is sent to establish the connection
type InitializeRequest struct {
	ProtocolVersion   string             `json:"protocolVersion"`
	Capabilities      ClientCapabilities `json:"capabilities"`
	ClientInfo        ClientInfo         `json:"clientInfo"`
}

// InitializeResponse is the server's response to initialize
type InitializeResponse struct {
	ProtocolVersion   string             `json:"protocolVersion"`
	Capabilities      ServerCapabilities `json:"capabilities"`
	ServerInfo        ServerInfo         `json:"serverInfo"`
	Instructions      string             `json:"instructions,omitempty"`
}

// ClientInfo contains information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Resource represents contextual data that can be provided to models
type Resource struct {
	URI         string      `json:"uri"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	MimeType    string      `json:"mimeType,omitempty"`
	Annotations *Annotation `json:"annotations,omitempty"`
}

// ResourceContents represents the actual content of a resource
type ResourceContents struct {
	URI      string           `json:"uri"`
	MimeType string           `json:"mimeType,omitempty"`
	Text     string           `json:"text,omitempty"`
	Blob     []byte           `json:"blob,omitempty"`
}

// Annotation provides additional metadata for resources
type Annotation struct {
	Audience []Role  `json:"audience,omitempty"`
	Priority float64 `json:"priority,omitempty"`
}

// Role defines who should see the resource
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Tool represents a function that can be called by the model
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema Schema      `json:"inputSchema"`
}

// Schema represents a JSON schema for tool parameters
type Schema struct {
	Type        string                 `json:"type"`
	Properties  map[string]Schema      `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Items       *Schema                `json:"items,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Additional  map[string]interface{} `json:"-"` // For additional properties
}

// CallToolRequest represents a tool call request
type CallToolRequest struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments,omitempty"`
}

// CallToolResponse represents a tool call response
type CallToolResponse struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents different types of content
type ContentItem struct {
	Type        string      `json:"type"`
	Text        string      `json:"text,omitempty"`
	Data        []byte      `json:"data,omitempty"`
	MimeType    string      `json:"mimeType,omitempty"`
	Annotations *Annotation `json:"annotations,omitempty"`
}

// Content types
const (
	ContentTypeText = "text"
	ContentTypeBlob = "blob"
)

// Prompt represents a templated prompt
type Prompt struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Arguments   []PromptArgument     `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt parameter
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// GetPromptRequest requests a specific prompt
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResponse returns the prompt result
type GetPromptResponse struct {
	Description string        `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    MessageRole   `json:"role"`
	Content []ContentItem `json:"content"`
}

// MessageRole defines the role in a conversation
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// ListResourcesResponse contains available resources
type ListResourcesResponse struct {
	Resources []Resource `json:"resources"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

// ListToolsResponse contains available tools  
type ListToolsResponse struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ListPromptsResponse contains available prompts
type ListPromptsResponse struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// ReadResourceRequest requests resource content
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResponse returns resource content
type ReadResourceResponse struct {
	Contents []ResourceContents `json:"contents"`
}

// MCPError represents MCP-specific errors
type MCPError struct {
	Code    int
	Message string
	Data    interface{}
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP Error %d: %s", e.Code, e.Message)
}

// Common MCP errors
var (
	ErrInvalidProtocolVersion = &MCPError{-32001, "Invalid protocol version", nil}
	ErrResourceNotFound       = &MCPError{-32002, "Resource not found", nil}
	ErrToolNotFound           = &MCPError{-32003, "Tool not found", nil}
	ErrPromptNotFound         = &MCPError{-32004, "Prompt not found", nil}
)

// MessageIDGenerator generates unique message IDs
type MessageIDGenerator struct {
	counter uint64
	mutex   sync.Mutex
}

func NewMessageIDGenerator() *MessageIDGenerator {
	return &MessageIDGenerator{}
}

func (g *MessageIDGenerator) Next() uint64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.counter++
	return g.counter
}

// CreateRequest creates a new JSON-RPC request message
func CreateRequest(id interface{}, method string, params interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// CreateResponse creates a new JSON-RPC response message
func CreateResponse(id interface{}, result interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// CreateErrorResponse creates a new JSON-RPC error response
func CreateErrorResponse(id interface{}, code int, message string, data interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// CreateNotification creates a JSON-RPC notification (no response expected)
func CreateNotification(method string, params interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// IsRequest checks if the message is a request
func (m *Message) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsResponse checks if the message is a response
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == "" && (m.Result != nil || m.Error != nil)
}

// IsNotification checks if the message is a notification
func (m *Message) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsError checks if the message is an error response
func (m *Message) IsError() bool {
	return m.Error != nil
}

// MarshalMessage marshals a message to JSON
func MarshalMessage(msg *Message) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalMessage unmarshals JSON to a message
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ValidateMessage validates a JSON-RPC message
func ValidateMessage(msg *Message) error {
	if msg.JSONRPC != "2.0" {
		return &MCPError{InvalidRequest, "Invalid JSON-RPC version", nil}
	}

	// Request validation
	if msg.IsRequest() {
		if msg.Method == "" {
			return &MCPError{InvalidRequest, "Request missing method", nil}
		}
		if msg.ID == nil {
			return &MCPError{InvalidRequest, "Request missing ID", nil}
		}
	}

	// Response validation  
	if msg.IsResponse() {
		if msg.ID == nil {
			return &MCPError{InvalidRequest, "Response missing ID", nil}
		}
		if msg.Result == nil && msg.Error == nil {
			return &MCPError{InvalidRequest, "Response missing result or error", nil}
		}
	}

	return nil
}