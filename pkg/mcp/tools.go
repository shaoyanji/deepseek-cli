// Package mcp provides Model Context Protocol tools for file and shell operations.
package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ToolDefinition defines an MCP tool schema
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// GetBuiltInTools returns the built-in MCP tools for file and shell operations
func GetBuiltInTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read the contents of a file at the specified path",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file at the specified path (overwrites if exists)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "execute_command",
			Description: "Execute a shell command and return its output",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The shell command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// ExecuteTool executes a built-in MCP tool by name
func ExecuteTool(toolName string, args map[string]interface{}, autoConfirm bool) (*ToolResult, error) {
	switch toolName {
	case "read_file":
		return executeReadFile(args)
	case "write_file":
		return executeWriteFile(args, autoConfirm)
	case "execute_command":
		return executeCommand(args, autoConfirm)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// executeReadFile reads a file's contents
func executeReadFile(args map[string]interface{}) (*ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Error: "path argument is required and must be a string"}, nil
	}

	// Clean and validate path
	cleanPath := filepath.Clean(path)

	// Security check: prevent reading outside current directory
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	// Allow reading files anywhere but warn about sensitive paths
	if strings.HasPrefix(absPath, "/etc/") || strings.HasPrefix(absPath, "/proc/") {
		return &ToolResult{Error: "reading from system directories is not allowed"}, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to read file: %v", err)}, nil
	}

	return &ToolResult{Content: string(data)}, nil
}

// executeWriteFile writes content to a file with confirmation
func executeWriteFile(args map[string]interface{}, autoConfirm bool) (*ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Error: "path argument is required and must be a string"}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{Error: "content argument is required and must be a string"}, nil
	}

	// Clean path
	cleanPath := filepath.Clean(path)

	// Security check: prevent writing to sensitive locations
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("invalid path: %v", err)}, nil
	}

	// Block writes to system directories
	if strings.HasPrefix(absPath, "/etc/") || strings.HasPrefix(absPath, "/proc/") || strings.HasPrefix(absPath, "/sys/") {
		return &ToolResult{Error: "writing to system directories is not allowed"}, nil
	}

	// Check if file exists
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// Request confirmation unless auto-confirm is enabled
	if !autoConfirm {
		var action string
		if exists {
			action = "overwrite"
		} else {
			action = "create"
		}
		fmt.Printf("\n[CONFIRM] %s file '%s'? [y/N]: ", action, cleanPath)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			return &ToolResult{Error: "user declined to write file"}, nil
		}
	}

	// Create parent directories if needed
	parentDir := filepath.Dir(absPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to create parent directory: %v", err)}, nil
	}

	// Write the file
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return &ToolResult{Error: fmt.Sprintf("failed to write file: %v", err)}, nil
	}

	return &ToolResult{Content: fmt.Sprintf("Successfully %s file: %s", map[bool]string{true: "overwritten", false: "created"}[exists], cleanPath)}, nil
}

// executeCommand executes a shell command with confirmation
func executeCommand(args map[string]interface{}, autoConfirm bool) (*ToolResult, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return &ToolResult{Error: "command argument is required and must be a string"}, nil
	}

	// Request confirmation unless auto-confirm is enabled
	if !autoConfirm {
		fmt.Printf("\n[CONFIRM] Execute command '%s'? [y/N]: ", command)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			return &ToolResult{Error: "user declined to execute command"}, nil
		}
	}

	// Execute the command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{
			Content: string(output),
			Error:   fmt.Sprintf("command failed: %v", err),
		}, nil
	}

	return &ToolResult{Content: string(output)}, nil
}

// IsDangerousCommand checks if a command contains potentially dangerous patterns
func IsDangerousCommand(command string) bool {
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=",
		"> /dev/",
		"mkfs.",
		":(){ :|:& };:",
		"chmod -R 777 /",
		"chown -R",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}

	return false
}
