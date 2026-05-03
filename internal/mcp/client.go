// Package mcp provides Model Context Protocol client functionality.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// TransportType represents the type of transport used
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

// ServerConfig holds configuration for an MCP server
type ServerConfig struct {
	Name      string        `toml:"name"`
	Command   string        `toml:"command"`
	Args      []string      `toml:"args"`
	URL       string        `toml:"url"`
	Transport TransportType `toml:"transport"`
	Timeout   time.Duration `toml:"timeout"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"serverName"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []ToolContent `json:"content"`
	Error   string        `json:"error,omitempty"`
}

// ToolContent represents content in a tool result
type ToolContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Client manages connections to MCP servers
type Client struct {
	mu       sync.Mutex
	servers  map[string]*serverConnection
	configs  []ServerConfig
	timeout  time.Duration
}

// serverConnection holds an active MCP server connection
type serverConnection struct {
	config    ServerConfig
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	bufReader *bufio.Reader
	tools     []Tool
	connected bool
	mu        sync.Mutex
}

// NewClient creates a new MCP client
func NewClient(configs []ServerConfig, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		servers: make(map[string]*serverConnection),
		configs: configs,
		timeout: timeout,
	}
}

// Connect connects to all configured servers
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cfg := range c.configs {
		if err := c.connectServer(ctx, cfg); err != nil {
			return fmt.Errorf("failed to connect to server %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// connectServer connects to a single MCP server
func (c *Client) connectServer(ctx context.Context, cfg ServerConfig) error {
	conn := &serverConnection{
		config:    cfg,
		connected: false,
	}

	switch cfg.Transport {
	case TransportStdio:
		if err := c.connectStdio(ctx, conn); err != nil {
			return err
		}
	case TransportSSE:
		if err := c.connectSSE(ctx, conn); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported transport: %s", cfg.Transport)
	}

	// Discover tools
	if err := c.discoverTools(ctx, conn); err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	c.servers[cfg.Name] = conn
	return nil
}

// connectStdio connects to an MCP server via stdio
func (c *Client) connectStdio(ctx context.Context, conn *serverConnection) error {
	cfg := conn.config
	
	// Expand environment variables in command and args
	cmdStr := os.ExpandEnv(cfg.Command)
	args := make([]string, len(cfg.Args))
	for i, arg := range cfg.Args {
		args[i] = os.ExpandEnv(arg)
	}

	cmd := exec.CommandContext(ctx, cmdStr, args...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	conn.cmd = cmd
	conn.stdin = stdin
	conn.stdout = stdout
	conn.bufReader = bufio.NewReader(stdout)
	conn.connected = true

	// Initialize the server
	if err := c.initializeStdio(ctx, conn); err != nil {
		cmd.Process.Kill()
		return err
	}

	return nil
}

// initializeStdio sends the initialize request to an stdio server
func (c *Client) initializeStdio(ctx context.Context, conn *serverConnection) error {
	initReq := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "deepseek-cli",
				"version": "1.0.0",
			},
		},
	}

	if err := c.sendStdioRequest(ctx, conn, initReq); err != nil {
		return err
	}

	var resp jsonrpcResponse
	if err := c.readStdioResponse(ctx, conn, &resp); err != nil {
		return err
	}

	// Send initialized notification
	notif := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	return c.sendStdioRequest(ctx, conn, notif)
}

// connectSSE connects to an MCP server via SSE
func (c *Client) connectSSE(ctx context.Context, conn *serverConnection) error {
	// For SSE, we'd typically establish an HTTP connection and listen for events
	// This is a simplified implementation
	conn.connected = true
	
	// Initialize via HTTP POST
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "deepseek-cli",
				"version": "1.0.0",
			},
		},
	}

	data, _ := json.Marshal(initReq)
	req, err := http.NewRequestWithContext(ctx, "POST", conn.config.URL+"/initialize", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// sendStdioRequest sends a JSON-RPC request via stdio
func (c *Client) sendStdioRequest(ctx context.Context, conn *serverConnection, req jsonrpcRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Set write deadline based on context if stdin supports it
	if deadline, ok := ctx.Deadline(); ok {
		if setter, ok := conn.stdin.(interface{ SetWriteDeadline(time.Time) error }); ok {
			setter.SetWriteDeadline(deadline)
			defer setter.SetWriteDeadline(time.Time{})
		}
	}

	_, err = conn.stdin.Write(data)
	if err != nil {
		return err
	}
	_, err = conn.stdin.Write([]byte("\n"))
	return err
}

// readStdioResponse reads a JSON-RPC response from stdio
func (c *Client) readStdioResponse(ctx context.Context, conn *serverConnection, resp *jsonrpcResponse) error {
	// Use the buffered reader for reading lines
	line, err := conn.bufReader.ReadBytes('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes.TrimSpace(line), resp)
}

// discoverTools discovers available tools from a server
func (c *Client) discoverTools(ctx context.Context, conn *serverConnection) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	switch conn.config.Transport {
	case TransportStdio:
		req := jsonrpcRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/list",
		}

		if err := c.sendStdioRequest(ctx, conn, req); err != nil {
			return err
		}

		var resp jsonrpcResponse
		if err := c.readStdioResponse(ctx, conn, &resp); err != nil {
			return err
		}

		if resp.Error != nil {
			return fmt.Errorf("tools/list error: %s", resp.Error.Message)
		}

		// Parse tools from response
		var toolsResp struct {
			Tools []Tool `json:"tools"`
		}
		if err := json.Unmarshal(resp.Result, &toolsResp); err != nil {
			return err
		}

		// Prefix tool names with server name
		for i := range toolsResp.Tools {
			toolsResp.Tools[i].ServerName = conn.config.Name
			toolsResp.Tools[i].Name = fmt.Sprintf("mcp__%s__%s", conn.config.Name, toolsResp.Tools[i].Name)
		}
		conn.tools = toolsResp.Tools

	case TransportSSE:
		// For SSE, make HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", conn.config.URL+"/tools/list", nil)
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: c.timeout}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var toolsResp struct {
			Tools []Tool `json:"tools"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
			return err
		}

		// Prefix tool names
		for i := range toolsResp.Tools {
			toolsResp.Tools[i].ServerName = conn.config.Name
			toolsResp.Tools[i].Name = fmt.Sprintf("mcp__%s__%s", conn.config.Name, toolsResp.Tools[i].Name)
		}
		conn.tools = toolsResp.Tools
	}

	return nil
}

// CallTool calls a tool on an MCP server
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find which server owns this tool
	var targetConn *serverConnection
	var actualToolName string

	for _, conn := range c.servers {
		conn.mu.Lock()
		for _, tool := range conn.tools {
			if tool.Name == toolName {
				targetConn = conn
				// Extract original tool name (remove prefix)
				parts := strings.SplitN(toolName, "__", 3)
				if len(parts) == 3 {
					actualToolName = parts[2]
				} else {
					actualToolName = toolName
				}
				break
			}
		}
		conn.mu.Unlock()
		
		if targetConn != nil {
			break
		}
	}

	if targetConn == nil {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	// Call the tool
	switch targetConn.config.Transport {
	case TransportStdio:
		return c.callToolStdio(ctx, targetConn, actualToolName, args)
	case TransportSSE:
		return c.callToolSSE(ctx, targetConn, actualToolName, args)
	default:
		return nil, fmt.Errorf("unsupported transport")
	}
}

// callToolStdio calls a tool via stdio transport
func (c *Client) callToolStdio(ctx context.Context, conn *serverConnection, toolName string, args map[string]interface{}) (*ToolResult, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	if err := c.sendStdioRequest(reqCtx, conn, req); err != nil {
		return nil, err
	}

	var resp jsonrpcResponse
	if err := c.readStdioResponse(reqCtx, conn, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return &ToolResult{Error: resp.Error.Message}, nil
	}

	var result ToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// callToolSSE calls a tool via SSE transport
func (c *Client) callToolSSE(ctx context.Context, conn *serverConnection, toolName string, args map[string]interface{}) (*ToolResult, error) {
	payload := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", conn.config.URL+"/tools/call", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTools returns all available tools from all servers
func (c *Client) GetTools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()

	var allTools []Tool
	for _, conn := range c.servers {
		conn.mu.Lock()
		allTools = append(allTools, conn.tools...)
		conn.mu.Unlock()
	}
	return allTools
}

// GetServerNames returns the names of all connected servers
func (c *Client) GetServerNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	names := make([]string, 0, len(c.servers))
	for name := range c.servers {
		names = append(names, name)
	}
	return names
}

// Close closes all server connections
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for _, conn := range c.servers {
		conn.mu.Lock()
		if conn.cmd != nil && conn.cmd.Process != nil {
			if err := conn.cmd.Process.Kill(); err != nil {
				lastErr = err
			}
		}
		if conn.stdin != nil {
			conn.stdin.Close()
		}
		conn.connected = false
		conn.mu.Unlock()
	}
	return lastErr
}

// IsConnected returns whether a specific server is connected
func (c *Client) IsConnected(serverName string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.servers[serverName]
	if !ok {
		return false
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.connected
}

// jsonrpcRequest represents a JSON-RPC 2.0 request
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcResponse represents a JSON-RPC 2.0 response
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError represents a JSON-RPC error
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
