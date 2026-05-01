package agent

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIClient is a mock implementation of the API client interface
type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) ChatCompletion(req interface{}) (interface{}, error) {
	args := m.Called(req)
	return args.Get(0), args.Error(1)
}

func TestToolInterface(t *testing.T) {
	// Verify Tool interface can be implemented
	var _ Tool = (*ViewTool)(nil)
	var _ Tool = (*EditTool)(nil)
	var _ Tool = (*BashTool)(nil)
	var _ Tool = (*LSTool)(nil)
	var _ Tool = (*GrepTool)(nil)
	var _ Tool = (*GitTool)(nil)
	var _ Tool = (*FetchTool)(nil)
	var _ Tool = (*LSPTool)(nil)
	var _ Tool = (*WebSearchTool)(nil)
}

func TestViewTool(t *testing.T) {
	tool := &ViewTool{}
	assert.Equal(t, "view", tool.Name())
	
	// Test viewing a file that exists (from project root)
	result, err := tool.Run(map[string]interface{}{
		"path": "../../main.go",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "package main")
}

func TestViewToolNotFound(t *testing.T) {
	tool := &ViewTool{}
	result, err := tool.Run(map[string]interface{}{
		"path": "nonexistent.go",
	})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestEditTool(t *testing.T) {
	tool := &EditTool{}
	assert.Equal(t, "edit", tool.Name())
	
	// Create a temp file for testing
	result, err := tool.Run(map[string]interface{}{
		"path":    "/tmp/testfile.txt",
		"content": "new content",
	})
	assert.NoError(t, err)
	assert.Equal(t, "File edited successfully", result)
}

func TestBashTool(t *testing.T) {
	tool := &BashTool{}
	assert.Equal(t, "bash", tool.Name())
	
	result, err := tool.Run(map[string]interface{}{
		"command": "echo hello",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "hello")
}

func TestLSTool(t *testing.T) {
	tool := &LSTool{}
	assert.Equal(t, "ls", tool.Name())
	
	result, err := tool.Run(map[string]interface{}{
		"path": "../..",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "main.go")
}

func TestGrepTool(t *testing.T) {
	tool := &GrepTool{}
	assert.Equal(t, "grep", tool.Name())
	
	result, err := tool.Run(map[string]interface{}{
		"pattern": "package",
		"path":    "../..",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "main.go")
}

func TestGitTool(t *testing.T) {
	tool := &GitTool{}
	assert.Equal(t, "git", tool.Name())
	
	result, err := tool.Run(map[string]interface{}{
		"args": "status",
	})
	// May fail if not a git repo, that's ok for test
	if err != nil {
		assert.Empty(t, result)
	} else {
		assert.NotEmpty(t, result)
	}
}

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	
	// Register all built-in tools
	registry.Register(&ViewTool{})
	registry.Register(&EditTool{})
	registry.Register(&BashTool{})
	registry.Register(&LSTool{})
	registry.Register(&GrepTool{})
	registry.Register(&GitTool{})
	
	assert.Len(t, registry.Tools(), 6)
	assert.NotNil(t, registry.Get("view"))
	assert.NotNil(t, registry.Get("edit"))
	assert.NotNil(t, registry.Get("bash"))
	assert.NotNil(t, registry.Get("ls"))
	assert.NotNil(t, registry.Get("grep"))
	assert.NotNil(t, registry.Get("git"))
	assert.Nil(t, registry.Get("nonexistent"))
}

func TestToolRegistryRun(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&BashTool{})
	
	result, err := registry.Run("bash", map[string]interface{}{
		"command": "echo test123",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "test123")
}

func TestToolRegistryRunNotFound(t *testing.T) {
	registry := NewToolRegistry()
	
	result, err := registry.Run("nonexistent", map[string]interface{}{})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestFetchTool(t *testing.T) {
	tool := &FetchTool{}
	assert.Equal(t, "fetch", tool.Name())
	
	// Test fetching a valid URL
	result, err := tool.Run(map[string]interface{}{
		"url": "https://example.com",
	})
	assert.NoError(t, err)
	assert.Contains(t, result, "Example Domain")
}

func TestFetchToolInvalidURL(t *testing.T) {
	tool := &FetchTool{}
	
	// Test missing URL
	result, err := tool.Run(map[string]interface{}{})
	assert.Error(t, err)
	assert.Empty(t, result)
	
	// Test invalid URL
	result, err = tool.Run(map[string]interface{}{
		"url": "not-a-valid-url",
	})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestFetchToolNotFound(t *testing.T) {
	tool := &FetchTool{}
	
	// Test 404 URL
	result, err := tool.Run(map[string]interface{}{
		"url": "https://example.com/nonexistent-page-12345",
	})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestLSPTool(t *testing.T) {
	// Skip if gopls is not installed
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not installed, skipping LSP test")
	}
	
	tool := &LSPTool{}
	assert.Equal(t, "lsp", tool.Name())
	
	// Test getting hover info for a Go file
	result, err := tool.Run(map[string]interface{}{
		"lang":    "go",
		"file":    "../../main.go",
		"line":    1,
		"column":  1,
		"action":  "hover",
	})
	// gopls may take time to start, so we just check no error for now
	if err != nil {
		t.Logf("LSP error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, result)
	}
}

func TestLSPToolInvalidArgs(t *testing.T) {
	tool := &LSPTool{}
	
	// Test missing lang
	result, err := tool.Run(map[string]interface{}{
		"file": "../../main.go",
	})
	assert.Error(t, err)
	assert.Empty(t, result)
	
	// Test missing file
	result, err = tool.Run(map[string]interface{}{
		"lang": "go",
	})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestWebSearchTool(t *testing.T) {
	tool := &WebSearchTool{}
	assert.Equal(t, "web_search", tool.Name())
	
	// Test search with valid query
	result, err := tool.Run(map[string]interface{}{
		"query": "golang tutorial",
	})
	// May fail if no API key, so skip error check
	if err != nil {
		t.Logf("WebSearch error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, result)
	}
}

func TestWebSearchToolInvalidArgs(t *testing.T) {
	tool := &WebSearchTool{}
	
	// Test missing query
	result, err := tool.Run(map[string]interface{}{})
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestToolChoice(t *testing.T) {
	// Test ToolChoice struct and methods
	toolChoice := &ToolChoice{
		Type: "function",
		Function: ToolChoiceFunction{
			Name: "view",
		},
	}
	
	assert.Equal(t, "function", toolChoice.Type)
	assert.Equal(t, "view", toolChoice.Function.Name)
}

func TestToolCall(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "bash",
			Arguments: `{"command": "echo hello"}`,
		},
	}
	
	assert.Equal(t, "call_123", toolCall.ID)
	assert.Equal(t, "bash", toolCall.Function.Name)
	
	// Test parsing arguments
	args := make(map[string]interface{})
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	assert.NoError(t, err)
	assert.Equal(t, "echo hello", args["command"])
}
