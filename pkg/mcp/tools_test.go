package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetBuiltInTools(t *testing.T) {
	tools := GetBuiltInTools()
	
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
	
	expectedNames := []string{"read_file", "write_file", "execute_command"}
	for i, name := range expectedNames {
		if tools[i].Name != name {
			t.Errorf("tool %d: expected name %s, got %s", i, name, tools[i].Name)
		}
	}
}

func TestExecuteReadFile(t *testing.T) {
	// Create a temp file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello world"
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	
	// Test reading the file
	result, err := executeReadFile(map[string]interface{}{"path": testFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Error != "" {
		t.Errorf("unexpected error in result: %s", result.Error)
	}
	
	if result.Content != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.Content)
	}
}

func TestExecuteReadFileNotFound(t *testing.T) {
	result, err := executeReadFile(map[string]interface{}{"path": "/nonexistent/file.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Error == "" {
		t.Error("expected error for nonexistent file")
	}
}

func TestExecuteWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")
	testContent := "new content"
	
	// Test writing a new file (with auto-confirm)
	result, err := executeWriteFile(map[string]interface{}{
		"path":    testFile,
		"content": testContent,
	}, true) // auto-confirm
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Error != "" {
		t.Errorf("unexpected error in result: %s", result.Error)
	}
	
	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}
}

func TestExecuteCommand(t *testing.T) {
	// Test a simple command (with auto-confirm)
	result, err := executeCommand(map[string]interface{}{
		"command": "echo hello",
	}, true) // auto-confirm
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Error != "" {
		t.Errorf("unexpected error in result: %s", result.Error)
	}
	
	if result.Content != "hello\n" {
		t.Errorf("expected content 'hello\\n', got %q", result.Content)
	}
}

func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		command  string
		dangerous bool
	}{
		{"rm -rf /", true},
		{"rm -rf /*", true},
		{"dd if=/dev/zero", true},
		{"echo hello", false},
		{"ls -la", false},
		{":(){ :|:& };:", true},
		{"chmod -R 777 /", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := IsDangerousCommand(tt.command)
			if result != tt.dangerous {
				t.Errorf("expected dangerous=%v, got %v", tt.dangerous, result)
			}
		})
	}
}
