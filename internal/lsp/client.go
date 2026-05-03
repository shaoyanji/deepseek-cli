// Package lsp provides Language Server Protocol client functionality for diagnostics.
package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Diagnostic represents a simplified diagnostic result
type Diagnostic struct {
	Severity    int    `json:"severity"`    // 1=Error, 2=Warning, 3=Info, 4=Hint
	Line        int    `json:"line"`        // 0-based line number
	Column      int    `json:"column"`      // 0-based column number
	EndLine     int    `json:"endLine"`     // 0-based end line
	EndColumn   int    `json:"endColumn"`   // 0-based end column
	Message     string `json:"message"`
	Source      string `json:"source"`
	Code        string `json:"code,omitempty"`
}

// ServerConfig holds configuration for an LSP server
type ServerConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	RootURI string   `toml:"root_uri"`
}

// Client manages LSP server connections and diagnostics
type Client struct {
	mu       sync.Mutex
	servers  map[string]*serverSession // keyed by language ID
	configs  map[string]ServerConfig   // keyed by language ID
	timeout  time.Duration
}

// serverSession holds an active LSP server session
type serverSession struct {
	cmd      *exec.Cmd
	stdin    *bufio.Writer
	stdout   *bufio.Reader
	seq      int64
	closed   bool
	done     chan struct{}
	diagnostics map[uri.URI][]protocol.Diagnostic
	mu       sync.Mutex
}

// NewClient creates a new LSP client with the given configurations
func NewClient(configs map[string]ServerConfig, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		servers: make(map[string]*serverSession),
		configs: configs,
		timeout: timeout,
	}
}

// RunDiagnostics runs diagnostics on a file and returns the results
func (c *Client) RunDiagnostics(ctx context.Context, filePath string) ([]Diagnostic, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Determine language from file extension
	langID := getLanguageID(filePath)
	if langID == "" {
		return nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(filePath))
	}

	// Get or start server for this language
	session, err := c.getOrCreateServer(ctx, langID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileURI := uri.File(filePath)

	// Send didOpen notification
	err = session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        fileURI,
			LanguageID: protocol.LanguageIdentifier(langID),
			Version:    1,
			Text:       string(content),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send didOpen: %w", err)
	}

	// Wait briefly for diagnostics to be published
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Request diagnostics
	session.mu.Lock()
	diags, ok := session.diagnostics[fileURI]
	session.mu.Unlock()

	if !ok {
		// Try calling textDocument/diagnostic directly
		var result interface{}
		err := session.call(ctx, "textDocument/diagnostic", map[string]interface{}{
			"textDocument": map[string]string{
				"uri": string(fileURI),
			},
		}, &result)
		
		if err != nil {
			// No diagnostics available yet, return empty
			return []Diagnostic{}, nil
		}

		// Parse result if possible
		diags = parseDiagnosticResult(result)
	}

	// Convert to our simplified format
	return convertDiagnostics(diags), nil
}

// Close shuts down all LSP servers
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for langID, session := range c.servers {
		session.close()
		delete(c.servers, langID)
	}
}

// getOrCreateServer gets an existing server session or creates a new one
func (c *Client) getOrCreateServer(ctx context.Context, langID, filePath string) (*serverSession, error) {
	if session, ok := c.servers[langID]; ok && !session.closed {
		return session, nil
	}

	config, ok := c.configs[langID]
	if !ok {
		// Try to auto-detect common servers
		config = autodetectServer(langID)
		if config.Command == "" {
			return nil, fmt.Errorf("no LSP server configured for language: %s", langID)
		}
	}

	// Determine root URI
	rootURI := config.RootURI
	if rootURI == "" {
		// Use directory of the file as root
		dir := filepath.Dir(filePath)
		if absDir, err := filepath.Abs(dir); err == nil {
			rootURI = string(uri.File(absDir))
		}
	}

	// Start the LSP server process
	cmd := exec.Command(config.Command, config.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	session := &serverSession{
		cmd:      cmd,
		stdin:    bufio.NewWriter(stdin),
		stdout:   bufio.NewReader(stdout),
		seq:      0,
		closed:   false,
		done:     make(chan struct{}),
		diagnostics: make(map[uri.URI][]protocol.Diagnostic),
	}

	// Initialize the LSP server
	initParams := protocol.InitializeParams{
		ProcessID:        int32(os.Getpid()),
		RootURI:          uri.URI(rootURI),
		Capabilities:     protocol.ClientCapabilities{},
		InitializationOptions: nil,
	}

	var initResult protocol.InitializeResult
	if err := session.call(ctx, "initialize", initParams, &initResult); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("LSP initialize failed: %w", err)
	}

	// Send initialized notification
	if err := session.sendNotification("initialized", struct{}{}); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("LSP initialized notification failed: %w", err)
	}

	// Start listening for diagnostics notifications in background
	go session.listenForDiagnostics()

	c.servers[langID] = session
	return session, nil
}

// sendNotification sends an LSP notification
func (s *serverSession) sendNotification(method string, params interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = s.stdin.Write(data)
	if err != nil {
		return err
	}
	_, err = s.stdin.Write([]byte("\r\n"))
	return err
}

// call makes an LSP request and waits for response
func (s *serverSession) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	s.mu.Lock()
	s.seq++
	id := s.seq
	s.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	s.mu.Lock()
	_, err = s.stdin.Write(data)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	_, err = s.stdin.Write([]byte("\r\n"))
	s.mu.Unlock()
	
	if err != nil {
		return err
	}

	// Wait for response
	respChan := make(chan *jsonrpcResponse, 1)
	go func() {
		resp := s.readResponse(id)
		respChan <- resp
	}()

	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return fmt.Errorf("LSP error: %s", resp.Error.Message)
		}
		if result != nil {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// readResponse reads responses until it finds the one matching the given ID
func (s *serverSession) readResponse(matchID int64) *jsonrpcResponse {
	for {
		line, err := s.stdout.ReadBytes('\n')
		if err != nil {
			return &jsonrpcResponse{
				Error: &jsonrpcError{Message: err.Error()},
			}
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		if resp.ID == matchID {
			return &resp
		}

		// Handle notifications (like publishDiagnostics)
		if resp.ID == 0 && resp.Method == "textDocument/publishDiagnostics" {
			s.handleDiagnosticsNotification(resp.Params)
		}
	}
}

// listenForDiagnostics listens for diagnostic notifications in the background
func (s *serverSession) listenForDiagnostics() {
	for {
		line, err := s.stdout.ReadBytes('\n')
		if err != nil {
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		if resp.Method == "textDocument/publishDiagnostics" {
			s.handleDiagnosticsNotification(resp.Params)
		}
	}
}

// handleDiagnosticsNotification processes a publishDiagnostics notification
func (s *serverSession) handleDiagnosticsNotification(params json.RawMessage) {
	var notif struct {
		URI         string                 `json:"uri"`
		Diagnostics []protocol.Diagnostic  `json:"diagnostics"`
	}

	if err := json.Unmarshal(params, &notif); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.diagnostics[uri.URI(notif.URI)] = notif.Diagnostics
}

// close shuts down the LSP server session
func (s *serverSession) close() {
	if s.closed {
		return
	}
	s.closed = true

	// Send shutdown
	s.sendNotification("shutdown", struct{}{})
	s.sendNotification("exit", struct{}{})

	// Kill the process
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}

	close(s.done)
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
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcError represents a JSON-RPC error
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// getLanguageID determines the language ID from file extension
func getLanguageID(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".js", ".jsx", ".ts", ".tsx":
		return "typescript"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	default:
		return ""
	}
}

// autodetectServer returns a default server config for common languages
func autodetectServer(langID string) ServerConfig {
	switch langID {
	case "go":
		return ServerConfig{Command: "gopls", Args: []string{"serve"}}
	case "python":
		return ServerConfig{Command: "pyright-langserver", Args: []string{"--stdio"}}
	case "rust":
		return ServerConfig{Command: "rust-analyzer"}
	case "typescript":
		return ServerConfig{Command: "typescript-language-server", Args: []string{"--stdio"}}
	default:
		return ServerConfig{}
	}
}

// convertDiagnostics converts protocol diagnostics to our simplified format
func convertDiagnostics(diags []protocol.Diagnostic) []Diagnostic {
	result := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		result = append(result, Diagnostic{
			Severity:  int(d.Severity),
			Line:      int(d.Range.Start.Line),
			Column:    int(d.Range.Start.Character),
			EndLine:   int(d.Range.End.Line),
			EndColumn: int(d.Range.End.Character),
			Message:   d.Message,
			Source:    d.Source,
			Code:      getStringCode(d.Code),
		})
	}
	return result
}

// parseDiagnosticResult attempts to parse a diagnostic result from various formats
func parseDiagnosticResult(result interface{}) []protocol.Diagnostic {
	// This is a simplified parser - real implementation would handle various formats
	// For now, return empty slice
	return []protocol.Diagnostic{}
}

// getStringCode extracts string code from CodeOrString type
func getStringCode(code interface{}) string {
	switch v := code.(type) {
	case string:
		return v
	case map[string]interface{}:
		if val, ok := v["value"].(string); ok {
			return val
		}
	}
	return ""
}
