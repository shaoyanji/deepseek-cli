package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Tool interface defines the contract for agent tools
type Tool interface {
	Name() string
	Run(args map[string]interface{}) (string, error)
}

// ToolChoice defines the model's selected tool
type ToolChoice struct {
	Type     string            `json:"type"`
	Function ToolChoiceFunction `json:"function"`
}

// ToolChoiceFunction represents the function selected by the model
type ToolChoiceFunction struct {
	Name string `json:"name"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents the function call details
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) Tool {
	return r.tools[name]
}

// Run executes a tool by name
func (r *ToolRegistry) Run(name string, args map[string]interface{}) (string, error) {
	tool := r.Get(name)
	if tool == nil {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return tool.Run(args)
}

// Tools returns all registered tools
func (r *ToolRegistry) Tools() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ViewTool implements file viewing
type ViewTool struct{}

func (t *ViewTool) Name() string {
	return "view"
}

func (t *ViewTool) Run(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument required")
	}
	
	// Clean the path for security
	cleanPath := filepath.Clean(path)
	
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	return string(data), nil
}

// EditTool implements file editing
type EditTool struct{}

func (t *EditTool) Name() string {
	return "edit"
}

func (t *EditTool) Run(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("path argument required")
	}
	
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument required")
	}
	
	// Create backup if file exists
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading file for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return "", fmt.Errorf("writing backup file: %w", err)
		}
	}
	
	// Write new content
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return "File edited successfully", nil
}

// BashTool implements command execution
type BashTool struct{}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Run(args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("command argument required")
	}
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}
	
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return stdout.String() + stderr.String(), err
	}
	
	return stdout.String(), nil
}

// LSTool implements directory listing
type LSTool struct{}

func (t *LSTool) Name() string {
	return "ls"
}

func (t *LSTool) Run(args map[string]interface{}) (string, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	
	var result strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		if info != nil {
			if info.IsDir() {
				result.WriteString("[DIR] ")
			} else {
				result.WriteString("      ")
			}
			result.WriteString(entry.Name())
			result.WriteString("\n")
		}
	}
	
	return result.String(), nil
}

// GrepTool implements content search
type GrepTool struct{}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Run(args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern argument required")
	}
	
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}
	
	// Use grep command if available, otherwise fallback to Go implementation
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Simple findstr fallback for Windows
		cmd = exec.Command("findstr", "/r", pattern, path+`\*`)
	} else {
		cmd = exec.Command("grep", "-r", pattern, path)
	}
	
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	_ = cmd.Run() // Ignore error as grep returns non-zero if no matches
	
	return stdout.String(), nil
}

// GitTool implements git commands
type GitTool struct{}

func (t *GitTool) Name() string {
	return "git"
}

func (t *GitTool) Run(args map[string]interface{}) (string, error) {
	gitArgs, ok := args["args"].(string)
	if !ok || gitArgs == "" {
		return "", fmt.Errorf("args argument required")
	}
	
	parts := strings.Fields(gitArgs)
	cmd := exec.Command("git", parts...)
	
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return stdout.String() + stderr.String(), err
	}
	
	return stdout.String(), nil
}

// FetchTool implements web content fetching
type FetchTool struct{}

func (t *FetchTool) Name() string {
	return "fetch"
}

func (t *FetchTool) Run(args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("url argument required")
	}
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	
	return string(body), nil
}

// LSPTool implements LSP integration for code context
type LSPTool struct{}

func (t *LSPTool) Name() string {
	return "lsp"
}

func (t *LSPTool) Run(args map[string]interface{}) (string, error) {
	lang, ok := args["lang"].(string)
	if !ok || lang == "" {
		return "", fmt.Errorf("lang argument required")
	}
	
	filePath, ok := args["file"].(string)
	if !ok || filePath == "" {
		return "", fmt.Errorf("file argument required")
	}
	
	// For Go, use gopls
	if lang == "go" {
		return t.queryGopls(filePath, args)
	}
	
	return "", fmt.Errorf("unsupported language for LSP: %s", lang)
}

func (t *LSPTool) queryGopls(file string, args map[string]interface{}) (string, error) {
	// Check if gopls is installed
	if _, err := exec.LookPath("gopls"); err != nil {
		return "", fmt.Errorf("gopls not installed: %w", err)
	}
	
	action := args["action"].(string)
	line, _ := args["line"].(int)
	column, _ := args["column"].(int)
	
	// Simplified: use gopls query command (this is a basic implementation)
	// Real LSP integration would use JSON-RPC over stdin/stdout
	switch action {
	case "hover":
		// For hover, we can use gopls's query functionality
		// This is a simplified version; full implementation would use LSP protocol
		cmd := exec.Command("gopls", "query", "hover", file, fmt.Sprintf("%d:%d", line, column))
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			return stdout.String() + stderr.String(), err
		}
		return stdout.String(), nil
	default:
		return "", fmt.Errorf("unsupported LSP action: %s", action)
	}
}

// WebSearchTool implements web search via Exa API
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Run(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query argument required")
	}
	
	apiKey := os.Getenv("EXA_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("EXA_API_KEY environment variable not set")
	}
	
	endpoint := "https://api.exa.ai/search"
	
	reqBody := map[string]interface{}{
		"query":       query,
		"numResults":  5,
		"useAutoprompt": true,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Exa API request failed with status: %s", resp.Status)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	return string(body), nil
}
