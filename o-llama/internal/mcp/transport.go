// Package mcp provides transport implementations for MCP protocol
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// StdioTransport implements MCP transport over stdio
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	connected bool
	mutex     sync.RWMutex
}

// NewStdioTransport creates a new stdio transport for a command
func NewStdioTransport(command string, args ...string) *StdioTransport {
	cmd := exec.Command(command, args...)
	
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	
	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: bufio.NewScanner(stdout),
	}
}

// Connect starts the command and establishes the transport
func (t *StdioTransport) Connect(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.connected {
		return fmt.Errorf("transport already connected")
	}
	
	// Start the command
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	t.connected = true
	
	// Monitor process in background
	go func() {
		err := t.cmd.Wait()
		t.mutex.Lock()
		t.connected = false
		t.mutex.Unlock()
		if err != nil && ctx.Err() == nil {
			// Process died unexpectedly
			fmt.Printf("MCP server process died: %v\n", err)
		}
	}()
	
	return nil
}

// Send sends a message over stdio
func (t *StdioTransport) Send(message *Message) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return fmt.Errorf("transport not connected")
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Write message with newline delimiter
	_, err = t.stdin.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	
	return nil
}

// Receive receives a message from stdio
func (t *StdioTransport) Receive() (*Message, error) {
	t.mutex.RLock()
	connected := t.connected
	t.mutex.RUnlock()
	
	if !connected {
		return nil, fmt.Errorf("transport not connected")
	}
	
	// Read line from stdout
	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read from stdout: %w", err)
		}
		return nil, fmt.Errorf("EOF from server")
	}
	
	line := t.scanner.Text()
	if line == "" {
		return t.Receive() // Skip empty lines
	}
	
	// Parse JSON message
	var message Message
	if err := json.Unmarshal([]byte(line), &message); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	
	return &message, nil
}

// IsConnected returns whether the transport is connected
func (t *StdioTransport) IsConnected() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.connected
}

// Close closes the transport and terminates the command
func (t *StdioTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if !t.connected {
		return nil
	}
	
	// Close stdin to signal shutdown to server
	if t.stdin != nil {
		t.stdin.Close()
	}
	
	// Give the process time to shutdown gracefully
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()
	
	select {
	case err := <-done:
		// Process exited gracefully
		t.connected = false
		return err
	case <-time.After(5 * time.Second):
		// Force kill the process
		if t.cmd.Process != nil {
			t.cmd.Process.Signal(syscall.SIGTERM)
			
			// Wait a bit more, then force kill
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.cmd.Process.Kill()
			}
		}
		t.connected = false
		return nil
	}
}

// SocketTransport implements MCP transport over Unix domain sockets or TCP
type SocketTransport struct {
	network   string // "unix" or "tcp"
	address   string
	conn      net.Conn
	encoder   *json.Encoder
	decoder   *json.Decoder
	connected bool
	mutex     sync.RWMutex
}

// NewSocketTransport creates a new socket transport
func NewSocketTransport(network, address string) *SocketTransport {
	return &SocketTransport{
		network: network,
		address: address,
	}
}

// Connect establishes the socket connection
func (t *SocketTransport) Connect(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.connected {
		return fmt.Errorf("transport already connected")
	}
	
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	
	conn, err := dialer.DialContext(ctx, t.network, t.address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s://%s: %w", t.network, t.address, err)
	}
	
	t.conn = conn
	t.encoder = json.NewEncoder(conn)
	t.decoder = json.NewDecoder(conn)
	t.connected = true
	
	return nil
}

// Send sends a message over the socket
func (t *SocketTransport) Send(message *Message) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return fmt.Errorf("transport not connected")
	}
	
	return t.encoder.Encode(message)
}

// Receive receives a message from the socket
func (t *SocketTransport) Receive() (*Message, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return nil, fmt.Errorf("transport not connected")
	}
	
	var message Message
	if err := t.decoder.Decode(&message); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}
	
	return &message, nil
}

// IsConnected returns whether the transport is connected
func (t *SocketTransport) IsConnected() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.connected
}

// Close closes the socket connection
func (t *SocketTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if !t.connected {
		return nil
	}
	
	err := t.conn.Close()
	t.connected = false
	return err
}

// HTTPTransport implements MCP transport over HTTP with Server-Sent Events
type HTTPTransport struct {
	baseURL   string
	client    *http.Client
	connected bool
	mutex     sync.RWMutex
	// SSE implementation would go here
}


// InMemoryTransport implements an in-memory transport for testing
type InMemoryTransport struct {
	incoming  chan *Message
	outgoing  chan *Message
	connected bool
	mutex     sync.RWMutex
}

// NewInMemoryTransport creates a new in-memory transport
func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{
		incoming: make(chan *Message, 100),
		outgoing: make(chan *Message, 100),
	}
}

// Connect establishes the in-memory connection
func (t *InMemoryTransport) Connect(ctx context.Context) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if t.connected {
		return fmt.Errorf("transport already connected")
	}
	
	t.connected = true
	return nil
}

// Send sends a message via the outgoing channel
func (t *InMemoryTransport) Send(message *Message) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return fmt.Errorf("transport not connected")
	}
	
	select {
	case t.outgoing <- message:
		return nil
	default:
		return fmt.Errorf("outgoing channel full")
	}
}

// Receive receives a message from the incoming channel
func (t *InMemoryTransport) Receive() (*Message, error) {
	t.mutex.RLock()
	connected := t.connected
	t.mutex.RUnlock()
	
	if !connected {
		return nil, fmt.Errorf("transport not connected")
	}
	
	select {
	case message := <-t.incoming:
		return message, nil
	default:
		return nil, fmt.Errorf("no messages available")
	}
}

// SendToIncoming sends a message to the incoming channel (for testing)
func (t *InMemoryTransport) SendToIncoming(message *Message) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return fmt.Errorf("transport not connected")
	}
	
	select {
	case t.incoming <- message:
		return nil
	default:
		return fmt.Errorf("incoming channel full")
	}
}

// ReceiveFromOutgoing receives a message from the outgoing channel (for testing)
func (t *InMemoryTransport) ReceiveFromOutgoing() (*Message, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if !t.connected {
		return nil, fmt.Errorf("transport not connected")
	}
	
	select {
	case message := <-t.outgoing:
		return message, nil
	default:
		return nil, fmt.Errorf("no messages available")
	}
}

// IsConnected returns whether the transport is connected
func (t *InMemoryTransport) IsConnected() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.connected
}

// Close closes the in-memory transport
func (t *InMemoryTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	if !t.connected {
		return nil
	}
	
	close(t.incoming)
	close(t.outgoing)
	t.connected = false
	return nil
}

// TransportFactory creates transports based on configuration
type TransportFactory struct{}

// CreateTransport creates a transport based on the given configuration
func (f *TransportFactory) CreateTransport(config map[string]interface{}) (Transport, error) {
	transportType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("transport type not specified")
	}
	
	switch transportType {
	case "stdio":
		command, ok := config["command"].(string)
		if !ok {
			return nil, fmt.Errorf("stdio transport requires command")
		}
		
		var args []string
		if argsInterface, exists := config["args"]; exists {
			if argsSlice, ok := argsInterface.([]interface{}); ok {
				for _, arg := range argsSlice {
					if argStr, ok := arg.(string); ok {
						args = append(args, argStr)
					}
				}
			}
		}
		
		return NewStdioTransport(command, args...), nil
		
	case "unix":
		path, ok := config["path"].(string)
		if !ok {
			return nil, fmt.Errorf("unix transport requires path")
		}
		
		return NewSocketTransport("unix", path), nil
		
	case "tcp":
		host, ok := config["host"].(string)
		if !ok {
			return nil, fmt.Errorf("tcp transport requires host")
		}
		
		port, ok := config["port"].(int)
		if !ok {
			return nil, fmt.Errorf("tcp transport requires port")
		}
		
		address := fmt.Sprintf("%s:%d", host, port)
		return NewSocketTransport("tcp", address), nil
		
	case "memory":
		return NewInMemoryTransport(), nil
		
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}